<script lang="ts">
	import { cartCount } from '$lib/stores/cart';
	import { authStore } from '$lib/stores/auth';
	import { goto } from '$app/navigation';

	async function handleLogout() {
		authStore.logout();
		goto('/');
	}
</script>

<header class="sticky top-0 z-50 bg-white border-b border-gray-200 shadow-sm">
	<div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8">
		<div class="flex items-center justify-between h-16">
			<!-- Logo -->
			<a href="/" class="text-xl font-bold text-primary-700 tracking-tight">stoa</a>

			<!-- Nav -->
			<nav class="hidden md:flex items-center gap-6 text-sm font-medium text-gray-600">
				<a href="/" class="hover:text-primary-700 transition-colors">Produkte</a>
				<a href="/search" class="hover:text-primary-700 transition-colors">Suche</a>
			</nav>

			<!-- Right actions -->
			<div class="flex items-center gap-3">
				<!-- Search (mobile) -->
				<a href="/search" class="md:hidden p-2 rounded-full hover:bg-gray-100 text-gray-500">
					<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
					</svg>
				</a>

				<!-- Account -->
				{#if $authStore.user}
					<div class="relative group">
						<button class="flex items-center gap-1 text-sm font-medium text-gray-700 hover:text-primary-700 transition-colors">
							<svg xmlns="http://www.w3.org/2000/svg" class="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor">
								<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M16 7a4 4 0 11-8 0 4 4 0 018 0zM12 14a7 7 0 00-7 7h14a7 7 0 00-7-7z" />
							</svg>
							<span class="hidden sm:inline">{$authStore.user.email}</span>
						</button>
						<div class="absolute right-0 top-full mt-1 w-44 bg-white border border-gray-200 rounded-lg shadow-lg opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all">
							<a href="/account/orders" class="block px-4 py-2 text-sm text-gray-700 hover:bg-gray-50">Meine Bestellungen</a>
							<button onclick={handleLogout} class="block w-full text-left px-4 py-2 text-sm text-gray-700 hover:bg-gray-50">Abmelden</button>
						</div>
					</div>
				{:else}
					<a href="/account/login" class="text-sm font-medium text-gray-600 hover:text-primary-700 transition-colors">Anmelden</a>
				{/if}

				<!-- Cart -->
				<a href="/cart" class="relative p-2 rounded-full hover:bg-gray-100 text-gray-700 transition-colors">
					<svg xmlns="http://www.w3.org/2000/svg" class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor">
						<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M3 3h2l.4 2M7 13h10l4-8H5.4M7 13L5.4 5M7 13l-2.293 2.293c-.63.63-.184 1.707.707 1.707H17m0 0a2 2 0 100 4 2 2 0 000-4zm-8 2a2 2 0 11-4 0 2 2 0 014 0z" />
					</svg>
					{#if $cartCount > 0}
						<span class="absolute -top-1 -right-1 h-5 w-5 rounded-full bg-primary-600 text-white text-xs flex items-center justify-center font-bold">
							{$cartCount}
						</span>
					{/if}
				</a>
			</div>
		</div>
	</div>
</header>
