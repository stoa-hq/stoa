import { writable } from 'svelte/store';

export interface UISelectOption {
	value: string;
	label: Record<string, string>;
}

export interface UISchemaField {
	key: string;
	type: 'text' | 'password' | 'toggle' | 'select' | 'number' | 'textarea';
	label: Record<string, string>;
	placeholder?: Record<string, string>;
	required?: boolean;
	options?: UISelectOption[];
	help_text?: Record<string, string>;
}

export interface UISchema {
	fields: UISchemaField[];
	submit_url?: string;
	load_url?: string;
}

export interface UIComponent {
	tag_name: string;
	script_url: string;
	integrity: string;
	external_scripts?: string[];
	style_url?: string;
}

export interface UIExtension {
	id: string;
	slot: string;
	type: 'schema' | 'component';
	schema?: UISchema;
	component?: UIComponent;
}

interface PluginManifest {
	extensions: UIExtension[];
	loaded: boolean;
}

const { subscribe, set } = writable<PluginManifest>({ extensions: [], loaded: false });

export const pluginStore = { subscribe };

export async function loadPluginManifest(): Promise<void> {
	try {
		const res = await fetch('/api/v1/store/plugin-manifest');
		if (!res.ok) return;
		const json = await res.json();
		set({ extensions: json.data?.extensions ?? [], loaded: true });
	} catch {
		set({ extensions: [], loaded: true });
	}
}
