<script lang="ts">
	import { authStore } from '$lib/stores/auth';
	import { theme } from '$lib/stores/theme';
	import { loadPluginManifest } from '$lib/stores/plugins';
	import { goto } from '$app/navigation';
	import { base } from '$app/paths';
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import Sidebar from '$lib/components/Layout/Sidebar.svelte';
	import Header from '$lib/components/Layout/Header.svelte';

	interface Props {
		children: import('svelte').Snippet;
	}
	let { children }: Props = $props();

	let ready = $state(false);
	let sidebarCollapsed = $state(false);
	let mobileSidebarOpen = $state(false);

	onMount(() => {
		if (!authStore.isAuthenticated()) {
			goto(`${base}/login`);
		} else {
			ready = true;
			theme.init();
			loadPluginManifest();
			const stored = localStorage.getItem('stoa_admin_sidebar');
			if (stored === 'collapsed') sidebarCollapsed = true;
		}
	});

	function toggleSidebar() {
		sidebarCollapsed = !sidebarCollapsed;
		localStorage.setItem('stoa_admin_sidebar', sidebarCollapsed ? 'collapsed' : 'expanded');
	}

	function toggleMobileSidebar() {
		mobileSidebarOpen = !mobileSidebarOpen;
	}
</script>

{#if ready}
	<div class="flex min-h-screen">
		<!-- Desktop Sidebar -->
		<div class="hidden md:block">
			<Sidebar collapsed={sidebarCollapsed} onToggle={toggleSidebar} />
		</div>

		<!-- Mobile Sidebar Overlay -->
		{#if mobileSidebarOpen}
			<div class="fixed inset-0 z-40 md:hidden" role="presentation">
				<button
					type="button"
					class="absolute inset-0 w-full h-full bg-black/60 backdrop-blur-sm cursor-default border-0 p-0"
					onclick={toggleMobileSidebar}
					aria-label="Close menu"
				></button>
				<div class="relative z-50 h-full w-56">
					<Sidebar collapsed={false} onToggle={toggleMobileSidebar} />
				</div>
			</div>
		{/if}

		<div class="flex-1 flex flex-col min-w-0">
			<Header title={$page.data?.title ?? 'Admin'} onMenuClick={toggleMobileSidebar} />
			<main class="flex-1 p-4 md:p-6 overflow-auto">
				{@render children()}
			</main>
		</div>
	</div>
{/if}
