declare module 'fzstd' {
  export function decompress(data: Uint8Array): Uint8Array;
  export function compress(data: Uint8Array, level?: number): Uint8Array;
}
