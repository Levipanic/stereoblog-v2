export function safeLink(value: string): string | null {
  if (!value || /[\s\\\u0000-\u001f\u007f]/.test(value) || value.startsWith('//')) return null
  try {
    const url = new URL(value, 'https://content.invalid')
    return ['http:', 'https:', 'mailto:'].includes(url.protocol) && !url.username && !url.password ? value : null
  }
  catch { return null }
}
