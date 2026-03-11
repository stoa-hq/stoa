<script lang="ts">
	import '$lib/i18n';
	import '../app.css';
	import Toast from '$lib/components/Toast.svelte';
	let { children } = $props();
</script>

{@render children()}
<Toast />
