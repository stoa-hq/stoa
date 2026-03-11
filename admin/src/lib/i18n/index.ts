import { register, init, getLocaleFromNavigator } from 'svelte-i18n';

register('de-DE', () => import('./de-DE.json'));
register('en-US', () => import('./en-US.json'));

const AVAILABLE = ['de-DE', 'en-US'];

function matchLocale(nav: string | null | undefined): string | undefined {
  if (!nav) return undefined;
  if (AVAILABLE.includes(nav)) return nav;
  const lang = nav.split('-')[0];
  return AVAILABLE.find(l => l.startsWith(lang));
}

const stored = typeof localStorage !== 'undefined' ? localStorage.getItem('stoa_admin_locale') : null;

init({
  fallbackLocale: 'de-DE',
  initialLocale: stored ?? matchLocale(getLocaleFromNavigator()) ?? 'de-DE',
});
