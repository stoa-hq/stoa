<script lang="ts">
	import '$lib/i18n';
	import '../app.css';
	import Toast from '$lib/components/Toast.svelte';
	import { isLoading } from 'svelte-i18n';
	let { children } = $props();
</script>

{#if $isLoading}
	<div class="min-h-screen flex items-center justify-center bg-gray-50">
		<div class="animate-spin h-8 w-8 border-4 border-primary-600 border-t-transparent rounded-full"></div>
	</div>
{:else}
	{@render children()}
	<Toast />
{/if}
