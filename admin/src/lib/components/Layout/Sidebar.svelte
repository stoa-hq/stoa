<script lang="ts">
	import { page } from '$app/stores';
	import { base } from '$app/paths';

	interface NavItem {
		href: string;
		label: string;
		icon: string;
	}

	const nav: NavItem[] = [
		{ href: '/', label: 'Dashboard', icon: '⊞' },
		{ href: '/products', label: 'Produkte', icon: '📦' },
		{ href: '/categories', label: 'Kategorien', icon: '🗂' },
		{ href: '/property-groups', label: 'Eigenschaften', icon: '🔧' },
		{ href: '/customers', label: 'Kunden', icon: '👥' },
		{ href: '/orders', label: 'Bestellungen', icon: '🛒' },
		{ href: '/media', label: 'Medien', icon: '🖼' },
		{ href: '/discounts', label: 'Rabatte', icon: '🏷' },
		{ href: '/tags', label: 'Tags', icon: '🔖' },
		{ href: '/tax', label: 'Steuern', icon: '📊' },
		{ href: '/shipping', label: 'Versand', icon: '🚚' },
		{ href: '/payment', label: 'Zahlungen', icon: '💳' },
		{ href: '/audit', label: 'Audit-Log', icon: '📋' }
	];

	function isActive(href: string): boolean {
		const fullHref = base + href;
		if (href === '/') return $page.url.pathname === fullHref || $page.url.pathname === base + '/';
		return $page.url.pathname.startsWith(fullHref);
	}
</script>

<aside class="w-56 shrink-0 bg-gray-900 min-h-screen flex flex-col">
	<div class="px-4 py-5 border-b border-gray-700">
		<span class="text-white font-bold text-lg">Commerce</span>
		<span class="text-gray-400 text-xs block">Admin Panel</span>
	</div>

	<nav class="flex-1 py-3 overflow-y-auto">
		{#each nav as item}
			<a
				href="{base}{item.href}"
				class="flex items-center gap-3 px-4 py-2 text-sm rounded-lg mx-2 my-0.5 transition-colors"
				class:bg-primary-600={isActive(item.href)}
				class:text-white={isActive(item.href)}
				class:text-gray-300={!isActive(item.href)}
				class:hover:bg-gray-800={!isActive(item.href)}
			>
				<span class="text-base">{item.icon}</span>
				{item.label}
			</a>
		{/each}
	</nav>
</aside>
