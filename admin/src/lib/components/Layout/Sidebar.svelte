<script lang="ts">
	import { page } from '$app/stores';
	import { base } from '$app/paths';
	import { t } from 'svelte-i18n';

	interface NavItem {
		href: string;
		labelKey: string;
		icon: string;
	}

	const nav: NavItem[] = [
		{ href: '/', labelKey: 'nav.dashboard', icon: '⊞' },
		{ href: '/products', labelKey: 'nav.products', icon: '📦' },
		{ href: '/categories', labelKey: 'nav.categories', icon: '🗂' },
		{ href: '/property-groups', labelKey: 'nav.propertyGroups', icon: '🔧' },
		{ href: '/customers', labelKey: 'nav.customers', icon: '👥' },
		{ href: '/orders', labelKey: 'nav.orders', icon: '🛒' },
		{ href: '/media', labelKey: 'nav.media', icon: '🖼' },
		{ href: '/discounts', labelKey: 'nav.discounts', icon: '🏷' },
		{ href: '/tags', labelKey: 'nav.tags', icon: '🔖' },
		{ href: '/tax', labelKey: 'nav.tax', icon: '📊' },
		{ href: '/shipping', labelKey: 'nav.shipping', icon: '🚚' },
		{ href: '/payment', labelKey: 'nav.payment', icon: '💳' },
		{ href: '/audit', labelKey: 'nav.audit', icon: '📋' }
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
				{$t(item.labelKey)}
			</a>
		{/each}
	</nav>
</aside>
