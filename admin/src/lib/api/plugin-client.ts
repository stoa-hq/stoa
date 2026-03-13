import { api } from '$lib/api/client';

const ALLOWED_PREFIXES = ['/admin/', '/plugins/'];

function isAllowedPath(path: string): boolean {
	if (path.includes('..')) return false;
	return ALLOWED_PREFIXES.some((prefix) => path.startsWith(prefix));
}

export function createPluginClient() {
	return {
		async get<T>(path: string): Promise<T> {
			if (!isAllowedPath(path)) throw new Error(`Plugin client: path not allowed: ${path}`);
			if (path.startsWith('/plugins/')) {
				const res = await fetch(path);
				if (!res.ok) throw new Error(`HTTP ${res.status}`);
				return res.json();
			}
			return api.get<T>(path);
		},
		async post<T>(path: string, body: unknown): Promise<T> {
			if (!isAllowedPath(path)) throw new Error(`Plugin client: path not allowed: ${path}`);
			if (path.startsWith('/plugins/')) {
				const res = await fetch(path, {
					method: 'POST',
					headers: { 'Content-Type': 'application/json' },
					body: JSON.stringify(body)
				});
				if (!res.ok) throw new Error(`HTTP ${res.status}`);
				return res.json();
			}
			return api.post<T>(path, body);
		}
	};
}
