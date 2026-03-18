<script lang="ts">
	import { t } from 'svelte-i18n';
	import { storeSettings } from '$lib/stores/settings';
</script>

<footer class="bg-gray-50 border-t border-gray-200 mt-16">
	<div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-10">
		<div class="flex flex-col md:flex-row items-center justify-between gap-4">
			<p class="text-sm text-gray-500">
				{#if $storeSettings.copyright_text}
					{$storeSettings.copyright_text}
				{:else}
					{$t('footer.copyright', { values: { year: new Date().getFullYear() } })}
				{/if}
				<span class="text-gray-400"> · powered by <a href="https://github.com/stoa-hq/stoa" target="_blank" rel="noopener noreferrer" class="hover:text-gray-600 transition-colors underline decoration-gray-300 hover:decoration-gray-500">stoa</a></span>
			</p>
			<nav class="flex items-center gap-6 text-sm text-gray-500">
				<a href="/" class="hover:text-gray-900 transition-colors">{$t('footer.products')}</a>
				<a href="/search" class="hover:text-gray-900 transition-colors">{$t('footer.search')}</a>
				<a href="/account/login" class="hover:text-gray-900 transition-colors">{$t('footer.account')}</a>
				<a href="/cart" class="hover:text-gray-900 transition-colors">{$t('footer.cart')}</a>
			</nav>
		</div>
	</div>
</footer>
