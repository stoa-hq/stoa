<script lang="ts">
	import { t } from 'svelte-i18n';
</script>

<footer class="bg-gray-50 border-t border-gray-200 mt-16">
	<div class="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-10">
		<div class="flex flex-col md:flex-row items-center justify-between gap-4">
			<p class="text-sm text-gray-500">{$t('footer.copyright', { values: { year: new Date().getFullYear() } })}</p>
			<nav class="flex items-center gap-6 text-sm text-gray-500">
				<a href="/" class="hover:text-gray-900 transition-colors">{$t('footer.products')}</a>
				<a href="/search" class="hover:text-gray-900 transition-colors">{$t('footer.search')}</a>
				<a href="/account/login" class="hover:text-gray-900 transition-colors">{$t('footer.account')}</a>
				<a href="/cart" class="hover:text-gray-900 transition-colors">{$t('footer.cart')}</a>
			</nav>
		</div>
	</div>
</footer>
