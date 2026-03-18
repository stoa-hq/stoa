import { writable } from 'svelte/store';

export interface StoreSettings {
	store_name: string;
	store_description: string;
	logo_url: string | null;
	favicon_url: string | null;
	contact_email: string | null;
	currency: string;
	country: string | null;
	timezone: string;
	copyright_text: string;
	maintenance_mode: boolean;
}

const defaults: StoreSettings = {
	store_name: 'Stoa',
	store_description: '',
	logo_url: null,
	favicon_url: null,
	contact_email: null,
	currency: 'EUR',
	country: null,
	timezone: 'UTC',
	copyright_text: '',
	maintenance_mode: false,
};

const { subscribe, set } = writable<StoreSettings>(defaults);

export const storeSettings = { subscribe };

export async function loadStoreSettings(): Promise<void> {
	try {
		const res = await fetch('/api/v1/store/settings');
		if (!res.ok) return;
		const json = await res.json();
		if (json.data) set(json.data);
	} catch {
		// keep defaults
	}
}
