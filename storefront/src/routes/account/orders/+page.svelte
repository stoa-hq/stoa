<script lang="ts">
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { authStore } from '$lib/stores/auth';
	import { ordersApi, type Order } from '$lib/api/orders';
	import { formatPrice, formatDate, orderStatusLabel, orderStatusClass } from '$lib/utils';

	let orders = $state<Order[]>([]);
	let loading = $state(true);
	let error = $state('');

	onMount(async () => {
		if (!authStore.isAuthenticated()) {
			goto('/account/login');
			return;
		}
		try {
			const res = await ordersApi.myOrders();
			orders = res.data ?? [];
		} catch {
			error = 'Bestellungen konnten nicht geladen werden.';
		} finally {
			loading = false;
		}
	});
</script>

<svelte:head>
	<title>Meine Bestellungen – stoa</title>
</svelte:head>

<div class="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-8">
	<div class="flex items-center justify-between mb-6">
		<h1 class="text-2xl font-bold text-gray-900">Meine Bestellungen</h1>
		<button onclick={() => { authStore.logout(); goto('/'); }} class="btn btn-secondary text-sm">Abmelden</button>
	</div>

	{#if loading}
		<div class="animate-pulse space-y-4">
			{#each Array(3) as _}
				<div class="h-20 bg-gray-100 rounded-xl"></div>
			{/each}
		</div>
	{:else if error}
		<p class="text-red-600">{error}</p>
	{:else if orders.length === 0}
		<div class="text-center py-20 text-gray-400">
			<p>Du hast noch keine Bestellungen aufgegeben.</p>
			<a href="/" class="btn btn-primary mt-4">Jetzt einkaufen</a>
		</div>
	{:else}
		<div class="space-y-4">
			{#each orders as order}
				<div class="card p-5">
					<div class="flex items-start justify-between gap-4">
						<div>
							<p class="font-semibold text-gray-900">#{order.order_number}</p>
							<p class="text-sm text-gray-500 mt-0.5">{formatDate(order.created_at)}</p>
						</div>
						<span class="badge {orderStatusClass(order.status)}">{orderStatusLabel(order.status)}</span>
					</div>
					{#if order.items && order.items.length > 0}
						<ul class="mt-3 space-y-1 text-sm text-gray-600">
							{#each order.items as item}
								<li class="flex justify-between">
									<span>{item.name} × {item.quantity}</span>
									<span>{formatPrice(item.total_gross)}</span>
								</li>
							{/each}
						</ul>
					{/if}
					<div class="flex justify-between items-center mt-3 pt-3 border-t border-gray-100">
						<span class="text-sm text-gray-500">Gesamt</span>
						<span class="font-bold text-gray-900">{formatPrice(order.total)}</span>
					</div>
				</div>
			{/each}
		</div>
	{/if}
</div>
