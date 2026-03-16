<script lang="ts">
	import { page } from '$app/stores';
	import { base } from '$app/paths';
	import { t } from 'svelte-i18n';
	import {
		LayoutDashboard, Package, FolderTree, Wrench, Users, ShoppingCart,
		Image, Tag, Bookmark, Receipt, Truck, CreditCard, ClipboardList,
		Warehouse,
		PanelLeftClose, PanelLeftOpen
	} from 'lucide-svelte';

	interface Props {
		collapsed: boolean;
		onToggle: () => void;
	}
	let { collapsed, onToggle }: Props = $props();

	const nav = [
		{ href: '/', labelKey: 'nav.dashboard', icon: LayoutDashboard },
		{ href: '/products', labelKey: 'nav.products', icon: Package },
		{ href: '/categories', labelKey: 'nav.categories', icon: FolderTree },
		{ href: '/property-groups', labelKey: 'nav.propertyGroups', icon: Wrench },
		{ href: '/customers', labelKey: 'nav.customers', icon: Users },
		{ href: '/orders', labelKey: 'nav.orders', icon: ShoppingCart },
		{ href: '/media', labelKey: 'nav.media', icon: Image },
		{ href: '/discounts', labelKey: 'nav.discounts', icon: Tag },
		{ href: '/tags', labelKey: 'nav.tags', icon: Bookmark },
		{ href: '/tax', labelKey: 'nav.tax', icon: Receipt },
		{ href: '/warehouses', labelKey: 'nav.warehouses', icon: Warehouse },
		{ href: '/shipping', labelKey: 'nav.shipping', icon: Truck },
		{ href: '/payment', labelKey: 'nav.payment', icon: CreditCard },
		{ href: '/audit', labelKey: 'nav.audit', icon: ClipboardList },
	];

	function isActive(href: string): boolean {
		const fullHref = base + href;
		if (href === '/') return $page.url.pathname === fullHref || $page.url.pathname === base + '/';
		return $page.url.pathname.startsWith(fullHref);
	}
</script>

<aside
	class="shrink-0 min-h-screen flex flex-col border-r border-[var(--sidebar-border)] bg-[var(--sidebar-bg)] transition-all duration-200
		{collapsed ? 'w-16' : 'w-56'}"
>
	<div class="px-4 py-5 border-b border-[var(--sidebar-border)] flex items-center {collapsed ? 'justify-center' : ''}">
		{#if !collapsed}
			<div>
				<span class="text-[var(--text)] font-bold text-lg">Stoa</span>
				<span class="text-[var(--text-muted)] text-xs block">Admin</span>
			</div>
		{:else}
			<span class="text-[var(--text)] font-bold text-lg">S</span>
		{/if}
	</div>

	<nav class="flex-1 py-3 overflow-y-auto">
		{#each nav as item}
			{@const active = isActive(item.href)}
			<a
				href="{base}{item.href}"
				class="flex items-center gap-3 px-3 py-2 text-sm rounded-lg mx-2 my-0.5 transition-all duration-150
					{collapsed ? 'justify-center px-0' : ''}
					{active
						? 'bg-primary-600/10 text-primary-600 dark:text-primary-400 font-medium'
						: 'text-[var(--text-muted)] hover:bg-gray-100 dark:hover:bg-white/5 hover:text-[var(--text)]'}"
				title={collapsed ? $t(item.labelKey) : undefined}
			>
				<item.icon class="w-[18px] h-[18px] shrink-0" />
				{#if !collapsed}
					<span class="truncate">{$t(item.labelKey)}</span>
				{/if}
			</a>
		{/each}
	</nav>

	<!-- Collapse Toggle (Desktop only) -->
	<div class="hidden md:block border-t border-[var(--sidebar-border)] p-2">
		<button
			class="flex items-center gap-2 w-full px-3 py-2 text-sm rounded-lg text-[var(--text-muted)] hover:text-[var(--text)] hover:bg-gray-100 dark:hover:bg-white/5 transition-colors
				{collapsed ? 'justify-center px-0' : ''}"
			onclick={onToggle}
		>
			{#if collapsed}
				<PanelLeftOpen class="w-[18px] h-[18px] shrink-0" />
			{:else}
				<PanelLeftClose class="w-[18px] h-[18px] shrink-0" />
				<span class="truncate">{$t('nav.collapse')}</span>
			{/if}
		</button>
	</div>
</aside>
