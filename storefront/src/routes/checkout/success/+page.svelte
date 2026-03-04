<script lang="ts">
	import { page } from '$app/stores';
	const orderNumber = $derived($page.url.searchParams.get('order') ?? '');
</script>

<svelte:head>
	<title>Bestellung erfolgreich – stoa</title>
</svelte:head>

<div class="max-w-lg mx-auto px-4 py-20 text-center">
	<div class="w-20 h-20 rounded-full bg-green-100 flex items-center justify-center mx-auto mb-6">
		<svg xmlns="http://www.w3.org/2000/svg" class="h-10 w-10 text-green-600" fill="none" viewBox="0 0 24 24" stroke="currentColor">
			<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
		</svg>
	</div>
	<h1 class="text-2xl font-bold text-gray-900">Vielen Dank für deine Bestellung!</h1>
	{#if orderNumber}
		<p class="text-gray-500 mt-2">Bestellnummer: <span class="font-medium text-gray-900">#{orderNumber}</span></p>
	{/if}
	<p class="text-gray-500 mt-4">Deine Bestellung wurde erfolgreich aufgegeben. Du erhältst in Kürze eine Bestätigung.</p>
	<div class="mt-8 flex flex-col sm:flex-row gap-3 justify-center">
		<a href="/account/orders" class="btn btn-primary">Meine Bestellungen</a>
		<a href="/" class="btn btn-secondary">Weiter einkaufen</a>
	</div>
</div>
