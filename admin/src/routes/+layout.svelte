<script lang="ts">
	import '../app.css';
	import Toast from '$lib/components/Toast.svelte';
	let { children } = $props();
</script>

{@render children()}
<Toast />
