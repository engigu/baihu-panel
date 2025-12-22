import pako from 'pako'

export function decompressFromBase64(compressed: string): string {
  if (!compressed) return ''
  try {
    // Decode base64 to binary
    const binaryString = atob(compressed)
    const bytes = new Uint8Array(binaryString.length)
    for (let i = 0; i < binaryString.length; i++) {
      bytes[i] = binaryString.charCodeAt(i)
    }
    // Decompress zlib
    const decompressed = pako.inflate(bytes, { to: 'string' })
    return decompressed
  } catch (e) {
    console.error('Decompress error:', e)
    return '[解压失败]'
  }
}
