const SDS_BASE = '/sds';

function encodePath(path) {
  return path
    .split('/')
    .filter(Boolean)
    .map((segment) => encodeURIComponent(segment))
    .join('/');
}

function fetchWithSignal(url, signal) {
  return signal ? fetch(url, { signal }) : fetch(url);
}

export async function getLocations({ signal } = {}) {
  const res = await fetchWithSignal(`${SDS_BASE}/fs`, signal);
  if (!res.ok) throw new Error(`Failed to load locations: ${res.status}`);
  const data = await res.json();
  return data.map((loc) => loc.location_name);
}

export async function getFiles(location, path = '', { signal } = {}) {
  const locationPath = encodeURIComponent(location);
  const filePath = encodePath(path);
  const url = filePath
    ? `${SDS_BASE}/fs/${locationPath}/${filePath}`
    : `${SDS_BASE}/fs/${locationPath}/`;
  const res = await fetchWithSignal(url, signal);
  if (!res.ok) throw new Error(`Failed to load files: ${res.status}`);
  return (await res.json()) ?? [];
}

export function getFileUrl(file, mode, location) {
  return `${SDS_BASE}/${mode}/${encodeURIComponent(location)}/${encodePath(file)}`;
}
