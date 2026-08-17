/**
 * Detects if the current user agent belongs to a Windows client.
 * This is used to apply system-level optimization styling or fonts.
 * @returns boolean
 */
export function isWindowsPlatform(): boolean {
  if (typeof navigator === 'undefined') return false
  return /Win/.test(navigator.userAgent)
}
