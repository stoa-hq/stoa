<script lang="ts">
	import { Search, X } from 'lucide-svelte';
	import { t } from 'svelte-i18n';

	interface Props {
		value: string;
		placeholder?: string;
		onSearch: (value: string) => void;
		debounce?: number;
	}
	let { value = $bindable(''), placeholder, onSearch, debounce = 400 }: Props = $props();

	let timeout: ReturnType<typeof setTimeout>;

	function handleInput() {
		clearTimeout(timeout);
		timeout = setTimeout(() => onSearch(value), debounce);
	}

	function clear() {
		value = '';
		clearTimeout(timeout);
		onSearch('');
	}
</script>

<div class="relative max-w-xs">
	<Search class="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400 dark:text-gray-500 pointer-events-none" />
	<input
		type="search"
		class="input pl-9 pr-8"
		bind:value
		oninput={handleInput}
		placeholder={placeholder ?? $t('common.search')}
	/>
	{#if value}
		<button
			type="button"
			class="absolute right-2 top-1/2 -translate-y-1/2 p-0.5 rounded text-gray-400 hover:text-gray-600 dark:hover:text-gray-300 transition-colors"
			onclick={clear}
		>
			<X class="w-3.5 h-3.5" />
		</button>
	{/if}
</div>
