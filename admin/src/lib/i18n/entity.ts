import { get } from 'svelte/store';
import { locale } from 'svelte-i18n';

/**
 * Resolves a translated field from an entity's translations array.
 * Tries the current svelte-i18n locale first, then falls back to 'de-DE',
 * then to the first available translation.
 */
// eslint-disable-next-line @typescript-eslint/no-explicit-any
export function tr(
	translations: Array<Record<string, any>> | undefined | null,
	field: string,
	currentLocale?: string | null
): string {
	if (!translations?.length) return '';
	const loc = currentLocale ?? get(locale) ?? 'de-DE';
	const match = translations.find((t) => t.locale === loc);
	if (match && match[field]) return String(match[field]);
	// Fallback to de-DE
	const fallback = translations.find((t) => t.locale === 'de-DE');
	if (fallback && fallback[field]) return String(fallback[field]);
	// Last resort: first available
	const first = translations[0];
	return first?.[field] ? String(first[field]) : '';
}
