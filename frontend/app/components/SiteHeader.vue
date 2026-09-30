<script setup lang="ts">
const config = useRuntimeConfig()
const { t, language, theme, feedView, setLanguage, setTheme, setFeedView } = useReaderSettings()
const settings = useTemplateRef<HTMLDetailsElement>('settings')

function closeSettings() {
  if (settings.value?.open) {
    settings.value.open = false
    settings.value.querySelector('summary')?.focus()
  }
}
</script>

<template>
  <a class="skip-link" href="#main-content">{{ t('skipToContent') }}</a>
  <header class="topbar">
    <div class="topbar-inner">
      <NuxtLink class="brand-mark" to="/" :aria-label="`${config.public.siteName} — ${t('home')}`">
        {{ config.public.siteName }}
      </NuxtLink>
      <details ref="settings" class="reader-settings" @keydown.esc.prevent="closeSettings">
        <summary>{{ t('settings') }}</summary>
        <div class="settings-panel">
          <fieldset>
            <legend>{{ t('language') }}</legend>
            <div class="setting-options">
              <button type="button" lang="ru" :aria-pressed="language === 'ru'" @click="setLanguage('ru')">Русский</button>
              <button type="button" lang="en" :aria-pressed="language === 'en'" @click="setLanguage('en')">English</button>
            </div>
          </fieldset>
          <fieldset>
            <legend>{{ t('theme') }}</legend>
            <div class="setting-options">
              <button type="button" :aria-pressed="theme === 'light'" @click="setTheme('light')">{{ t('light') }}</button>
              <button type="button" :aria-pressed="theme === 'dark'" @click="setTheme('dark')">{{ t('dark') }}</button>
            </div>
          </fieldset>
          <fieldset>
            <legend>{{ t('feedView') }}</legend>
            <div class="setting-options">
              <button type="button" :aria-pressed="feedView === 'list'" @click="setFeedView('list')">{{ t('feedList') }}</button>
              <button type="button" :aria-pressed="feedView === 'grid'" @click="setFeedView('grid')">{{ t('feedGrid') }}</button>
            </div>
          </fieldset>
        </div>
      </details>
    </div>
  </header>
</template>
