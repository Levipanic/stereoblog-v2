package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Levipanic/stereoblog-v2/backend/internal/antispam"
	"github.com/Levipanic/stereoblog-v2/backend/internal/comments"
	"github.com/Levipanic/stereoblog-v2/backend/internal/config"
	"github.com/Levipanic/stereoblog-v2/backend/internal/posts"
)

const (
	defaultCommentBodyLimit = 256 << 10
	commentSoftLimit        = "Too many messages in a row. Please try a bit later."
)

type commentCreateResponse struct {
	OK     bool   `json:"ok"`
	Status string `json:"status"`
}

func commentCreateHandler(service *antispam.Service, repository *comments.Repository, limiter *fixedWindowLimiter, cfg config.Config, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !isJSONRequest(c) {
			writeError(c, http.StatusUnsupportedMediaType, "invalid_content_type", "Unsupported content type. Please send JSON.")
			return
		}
		if allowed, retryAfter := limiter.allow(ClientIP(c)); !allowed {
			setRetryAfter(c, retryAfter)
			writeError(c, http.StatusTooManyRequests, "comment_rate_limited", commentSoftLimit)
			return
		}
		bodyLimit := cfg.HTTP.JSONBodyLimit
		if bodyLimit <= 0 {
			bodyLimit = defaultCommentBodyLimit
		}
		rawBody, err := io.ReadAll(http.MaxBytesReader(c.Writer, c.Request.Body, bodyLimit))
		if err != nil || len(rawBody) == 0 {
			writeError(c, http.StatusBadRequest, "invalid_body", "Request body is invalid or too large.")
			return
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(rawBody, &fields); err != nil {
			writeError(c, http.StatusBadRequest, "invalid_body", "Request body must be valid JSON.")
			return
		}
		name := jsonFieldText(fields["name"])
		content := jsonFieldText(fields["content"])
		website := jsonFieldText(fields["website"])
		challengeToken := jsonFieldText(fields["challenge_token"])

		postID, err := strconv.ParseInt(c.Param("post"), 10, 64)
		validPost := err == nil && postID > 0
		var postIDRef *int64
		if validPost {
			postIDRef = &postID
		}
		ipHash := posts.HashIP(cfg.Likes.IPHashSalt, ClientIP(c))

		mute, err := service.ActiveMute(c.Request.Context(), ipHash)
		if err != nil {
			logger.Error("comment mute lookup failed", "error", err)
			writeError(c, http.StatusInternalServerError, "internal_error", "Internal server error.")
			return
		}
		if mute != nil {
			_ = service.RecordAttempt(c.Request.Context(), antispam.Attempt{
				IPHash: ipHash, PostID: postIDRef, Status: antispam.Muted, Reason: mute.Reason, Content: content,
			})
			setRetryAfter(c, mute.RetryAfter)
			writeError(c, http.StatusTooManyRequests, "comment_rate_limited", commentSoftLimit)
			return
		}

		if !validPost {
			_ = service.RecordAttempt(c.Request.Context(), antispam.Attempt{
				IPHash: ipHash, PostID: nil, Status: antispam.Rejected, Reason: "invalid_post_id", Content: content,
			})
			writeError(c, http.StatusBadRequest, "invalid_post_id", "Post ID must be a positive integer.")
			return
		}

		verified, err := service.VerifyChallenge(challengeToken, postID)
		if err != nil {
			reason := "invalid_challenge"
			var challengeErr *antispam.ChallengeError
			if errors.As(err, &challengeErr) {
				reason = challengeErr.Reason
			}
			_ = service.RecordAttempt(c.Request.Context(), antispam.Attempt{
				IPHash: ipHash, PostID: postIDRef, Status: antispam.Rejected, Reason: reason, Content: content,
			})
			writeError(c, http.StatusTooManyRequests, "comment_rate_limited", commentSoftLimit)
			return
		}

		if website != "" || jsonFieldText(fields[verified.HoneypotField]) != "" {
			mute, err := service.RecordHoneypot(c.Request.Context(), antispam.Attempt{
				IPHash: ipHash, PostID: postIDRef, Content: content,
			})
			if err != nil {
				logger.Error("comment honeypot recording failed", "error", err)
				writeError(c, http.StatusInternalServerError, "internal_error", "Internal server error.")
				return
			}
			if mute != nil {
				setRetryAfter(c, mute.RetryAfter)
			}
			writeError(c, http.StatusTooManyRequests, "comment_rate_limited", commentSoftLimit)
			return
		}

		if antispam.UTF16Length(name) > cfg.Comments.MaxNameLength {
			_ = service.RecordAttempt(c.Request.Context(), antispam.Attempt{
				IPHash: ipHash, PostID: postIDRef, Status: antispam.Rejected, Reason: "name_too_long", Content: content,
			})
			writeError(c, http.StatusBadRequest, "invalid_name",
				"Comment name is too long. Maximum is "+strconv.Itoa(cfg.Comments.MaxNameLength)+" characters.")
			return
		}

		validatedContent, err := service.ValidateContent(content)
		if err != nil {
			_ = service.RecordAttempt(c.Request.Context(), antispam.Attempt{
				IPHash: ipHash, PostID: postIDRef, Status: antispam.Rejected, Reason: "validation:" + err.Error(), Content: content,
			})
			var validationErr *antispam.ValidationError
			if errors.As(err, &validationErr) && validationErr.Mask {
				writeError(c, http.StatusTooManyRequests, "comment_rate_limited", commentSoftLimit)
				return
			}
			writeError(c, http.StatusBadRequest, "invalid_content", err.Error())
			return
		}
		content = validatedContent
		textHash := antispam.TextHash(content)
		fingerprint := antispam.Fingerprint(content)

		parentID, hasParent, parentValid := jsonFieldID(fields["parent_id"])
		if hasParent && !parentValid {
			_ = service.RecordAttempt(c.Request.Context(), antispam.Attempt{
				IPHash: ipHash, PostID: postIDRef, Status: antispam.Rejected, Reason: "invalid_parent_comment_id",
				Content: content, TextHash: textHash, Fingerprint: fingerprint,
			})
			writeError(c, http.StatusBadRequest, "invalid_parent_id", "Invalid parent comment id.")
			return
		}
		var parentRef *int64
		if hasParent {
			parentRef = &parentID
		}

		exists, err := repository.PostExists(c.Request.Context(), postID)
		if err != nil {
			logger.Error("comment post lookup failed", "error", err)
			writeError(c, http.StatusInternalServerError, "internal_error", "Internal server error.")
			return
		}
		if !exists {
			_ = service.RecordAttempt(c.Request.Context(), antispam.Attempt{
				IPHash: ipHash, PostID: postIDRef, Status: antispam.Rejected, Reason: "post_not_found",
				Content: content, TextHash: textHash, Fingerprint: fingerprint,
			})
			writeError(c, http.StatusNotFound, "post_not_found", "Post not found.")
			return
		}

		if hasParent {
			parentOK, err := repository.ParentForPost(c.Request.Context(), parentID, postID)
			if err != nil {
				logger.Error("comment parent lookup failed", "error", err)
				writeError(c, http.StatusInternalServerError, "internal_error", "Internal server error.")
				return
			}
			if !parentOK {
				_ = service.RecordAttempt(c.Request.Context(), antispam.Attempt{
					IPHash: ipHash, PostID: postIDRef, Status: antispam.Rejected, Reason: "parent_comment_not_found",
					Content: content, TextHash: textHash, Fingerprint: fingerprint,
				})
				writeError(c, http.StatusNotFound, "parent_comment_not_found", "Parent comment not found.")
				return
			}
		}

		if err := service.ConsumeChallenge(c.Request.Context(), verified); err != nil {
			var challengeErr *antispam.ChallengeError
			if !errors.As(err, &challengeErr) {
				logger.Error("comment challenge consumption failed", "error", err)
				writeError(c, http.StatusInternalServerError, "internal_error", "Internal server error.")
				return
			}
			_ = service.RecordAttempt(c.Request.Context(), antispam.Attempt{
				IPHash: ipHash, PostID: postIDRef, Status: antispam.Rejected, Reason: challengeErr.Reason,
				Content: content, TextHash: textHash, Fingerprint: fingerprint,
			})
			writeError(c, http.StatusTooManyRequests, "comment_rate_limited", commentSoftLimit)
			return
		}

		stats, err := service.AttemptStats(c.Request.Context(), ipHash, postID, textHash, fingerprint)
		if err != nil {
			logger.Error("comment attempt stats failed", "error", err)
			writeError(c, http.StatusInternalServerError, "internal_error", "Internal server error.")
			return
		}
		rate := service.RateLimit(stats)
		if rate.Limited {
			mute, err := service.RecordRateRejection(c.Request.Context(), antispam.Attempt{
				IPHash: ipHash, PostID: postIDRef, Status: antispam.Rejected, Reason: "rate:" + rate.Reason,
				Content: content, TextHash: textHash, Fingerprint: fingerprint,
			}, rate)
			if err != nil {
				logger.Error("comment rate rejection recording failed", "error", err)
				writeError(c, http.StatusInternalServerError, "internal_error", "Internal server error.")
				return
			}
			retryAfter := rate.RetryAfter
			if mute != nil {
				retryAfter = mute.RetryAfter
			}
			setRetryAfter(c, retryAfter)
			writeError(c, http.StatusTooManyRequests, "comment_rate_limited", commentSoftLimit)
			return
		}

		decision := service.Moderate(antispam.RequestSource{
			Host: c.Request.Host, Origin: c.GetHeader("Origin"), Referer: c.GetHeader("Referer"),
		}, name, content, stats)

		var nameRef *string
		if name != "" {
			nameRef = &name
		}
		if _, err := repository.Create(c.Request.Context(), comments.CreateInput{
			PostID: postID, ParentID: parentRef, Name: nameRef, Content: content,
			Status: decision.Status, ModerationReason: decision.Reason, TextHash: textHash, Fingerprint: fingerprint,
		}); err != nil {
			logger.Error("comment insert failed", "error", err)
			writeError(c, http.StatusInternalServerError, "internal_error", "Internal server error.")
			return
		}
		if err := service.RecordAttempt(c.Request.Context(), antispam.Attempt{
			IPHash: ipHash, PostID: postIDRef, Status: decision.Status, Reason: decision.Reason,
			Content: content, TextHash: textHash, Fingerprint: fingerprint,
		}); err != nil {
			logger.Error("comment attempt record failed", "error", err)
		}
		status := http.StatusCreated
		if decision.Status == antispam.Pending {
			status = http.StatusAccepted
		}
		c.JSON(status, commentCreateResponse{OK: true, Status: decision.Status})
	}
}

func isJSONRequest(c *gin.Context) bool {
	contentType := strings.ToLower(c.GetHeader("Content-Type"))
	return strings.HasPrefix(contentType, "application/json") || strings.Contains(contentType, "+json")
}

func jsonFieldText(field json.RawMessage) string {
	if len(field) == 0 {
		return ""
	}
	var value string
	if json.Unmarshal(field, &value) != nil {
		return ""
	}
	return value
}

func jsonFieldID(field json.RawMessage) (id int64, present, valid bool) {
	value := strings.TrimSpace(string(field))
	if len(value) == 0 || value == "null" {
		return 0, false, true
	}
	if value[0] == '"' {
		var text string
		if json.Unmarshal(field, &text) != nil {
			return 0, true, false
		}
		value = strings.TrimSpace(text)
		if value == "" {
			return 0, false, true
		}
	}
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, true, false
	}
	return id, true, true
}
