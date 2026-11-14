// Base64 helpers (standard base64, as Go's encoding/base64 StdEncoding).
// b64decode also accepts base64url input when urlSafe is true (JWK
// fields use base64url without padding).

export function b64encode(bytes: Uint8Array): string {
  let s = "";
  for (let i = 0; i < bytes.length; i++) s += String.fromCharCode(bytes[i]);
  return btoa(s);
}

export function b64decode(s: string, urlSafe = false): Uint8Array {
  if (s === "") return new Uint8Array(0);
  let norm = s;
  if (urlSafe) norm = norm.replace(/-/g, "+").replace(/_/g, "/");
  while (norm.length % 4 !== 0) norm += "=";
  const bin = atob(norm);
  const out = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i++) out[i] = bin.charCodeAt(i);
  return out;
}
