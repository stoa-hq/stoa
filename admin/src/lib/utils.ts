/** Format cents to currency string: 1999 → "19,99 €" */
export function formatPrice(cents: number, currency = 'EUR'): string {
	return new Intl.NumberFormat('de-DE', {
		style: 'currency',
		currency
	}).format(cents / 100);
}

/** Format tax rate from basis points: 1900 → "19,00 %" */
export function formatTaxRate(basisPoints: number): string {
	return new Intl.NumberFormat('de-DE', {
		style: 'percent',
		minimumFractionDigits: 2
	}).format(basisPoints / 10000);
}

/** Format ISO date string to locale date */
export function formatDate(iso: string): string {
	if (!iso) return '–';
	return new Date(iso).toLocaleDateString('de-DE', {
		day: '2-digit',
		month: '2-digit',
		year: 'numeric'
	});
}

/** Format ISO date string to locale datetime */
export function formatDateTime(iso: string): string {
	if (!iso) return '–';
	return new Date(iso).toLocaleString('de-DE', {
		day: '2-digit',
		month: '2-digit',
		year: 'numeric',
		hour: '2-digit',
		minute: '2-digit'
	});
}

/** Format file size in bytes to human readable */
export function formatBytes(bytes: number): string {
	if (bytes < 1024) return `${bytes} B`;
	if (bytes < 1024 ** 2) return `${(bytes / 1024).toFixed(1)} KB`;
	return `${(bytes / 1024 ** 2).toFixed(1)} MB`;
}

/** Order status color helper */
export function orderStatusBadge(status: string): string {
	const map: Record<string, string> = {
		pending: 'badge-yellow',
		confirmed: 'badge-blue',
		processing: 'badge-blue',
		shipped: 'badge-blue',
		delivered: 'badge-green',
		completed: 'badge-green',
		cancelled: 'badge-red',
		refunded: 'badge-gray'
	};
	return map[status] ?? 'badge-gray';
}

/** Truncate a string */
export function truncate(s: string, max = 60): string {
	return s.length > max ? s.slice(0, max) + '…' : s;
}
