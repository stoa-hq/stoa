export function formatPrice(cents: number, currency = 'EUR', locale = 'de-DE'): string {
	return new Intl.NumberFormat(locale, {
		style: 'currency',
		currency
	}).format(cents / 100);
}

export function formatDate(dateStr: string, locale = 'de-DE'): string {
	return new Intl.DateTimeFormat(locale, {
		day: '2-digit',
		month: '2-digit',
		year: 'numeric'
	}).format(new Date(dateStr));
}

export function orderStatusLabel(status: string): string {
	const labels: Record<string, string> = {
		pending: 'Ausstehend',
		confirmed: 'Bestätigt',
		processing: 'In Bearbeitung',
		shipped: 'Versendet',
		delivered: 'Zugestellt',
		cancelled: 'Storniert',
		refunded: 'Erstattet'
	};
	return labels[status] ?? status;
}

export function orderStatusClass(status: string): string {
	const classes: Record<string, string> = {
		pending: 'badge-gray',
		confirmed: 'badge-green',
		processing: 'badge-green',
		shipped: 'badge-green',
		delivered: 'badge-green',
		cancelled: 'badge-red',
		refunded: 'badge-red'
	};
	return classes[status] ?? 'badge-gray';
}
