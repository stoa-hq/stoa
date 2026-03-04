export const AVAILABLE_LOCALES = ['de-DE', 'en-US'];
export const DEFAULT_LOCALE = 'de-DE';

export const LOCALE_LABELS: Record<string, string> = {
	'de-DE': 'Deutsch',
	'en-US': 'English',
};

/** Initializes a translations map with empty strings for all locales and fields. */
export function emptyTranslations(fields: string[]): Record<string, Record<string, string>> {
	return Object.fromEntries(
		AVAILABLE_LOCALES.map((l) => [l, Object.fromEntries(fields.map((f) => [f, '']))])
	);
}

/** Merges a translations array (from API) into an empty translations map. */
export function translationsFromArray<T extends { locale: string }>(
	arr: T[] | undefined,
	fields: string[]
): Record<string, Record<string, string>> {
	const result = emptyTranslations(fields);
	for (const t of arr ?? []) {
		if (result[t.locale]) {
			for (const field of fields) {
				result[t.locale][field] = (t as Record<string, string>)[field] ?? '';
			}
		}
	}
	return result;
}

/** Converts a translations map to an array for the API, skipping locales without a name. */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function translationsToArray(translations: Record<string, Record<string, string>>): any[] {
	return Object.entries(translations)
		.filter(([, t]) => t.name?.trim())
		.map(([locale, t]) => ({ locale, ...t }));
}
