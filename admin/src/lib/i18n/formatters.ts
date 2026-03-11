import { derived } from 'svelte/store';
import { locale } from 'svelte-i18n';

export const fmt = derived(locale, ($locale) => {
  const loc = $locale ?? 'de-DE';
  return {
    price(cents: number, currency = 'EUR'): string {
      return new Intl.NumberFormat(loc, { style: 'currency', currency }).format(cents / 100);
    },
    date(iso: string): string {
      return new Date(iso).toLocaleDateString(loc, {
        day: '2-digit',
        month: '2-digit',
        year: 'numeric',
      });
    },
    dateTime(iso: string): string {
      return new Date(iso).toLocaleString(loc, {
        day: '2-digit',
        month: '2-digit',
        year: 'numeric',
        hour: '2-digit',
        minute: '2-digit',
      });
    },
    taxRate(basisPoints: number): string {
      return new Intl.NumberFormat(loc, {
        style: 'percent',
        minimumFractionDigits: 2,
        maximumFractionDigits: 2,
      }).format(basisPoints / 10000);
    },
    number(value: number): string {
      return new Intl.NumberFormat(loc).format(value);
    },
  };
});
