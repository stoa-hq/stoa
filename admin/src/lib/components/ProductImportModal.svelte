<script lang="ts">
	import { fade, scale } from 'svelte/transition';
	import { productsApi } from '$lib/api/products';
	import type { BulkResponse } from '$lib/types';
	import { t } from 'svelte-i18n';

	interface Props {
		open: boolean;
		onClose: () => void;
		onImported: () => void;
	}

	let { open, onClose, onImported }: Props = $props();

	type Tab = 'csv' | 'json';
	type Phase = 'idle' | 'loading' | 'result';

	let activeTab = $state<Tab>('csv');
	let phase = $state<Phase>('idle');
	let result = $state<BulkResponse | null>(null);
	let errorMsg = $state('');

	// CSV state
	let csvFile = $state<File | null>(null);
	let dragOver = $state(false);
	let fileInput = $state<HTMLInputElement>(null!);

	// JSON state
	let jsonText = $state('');
	let jsonError = $state('');

	// Expand state per failed row
	let expanded = $state<Record<number, boolean>>({});

	function reset() {
		phase = 'idle';
		result = null;
		errorMsg = '';
		csvFile = null;
		jsonText = '';
		jsonError = '';
		expanded = {};
		activeTab = 'csv';
		dragOver = false;
	}

	function handleClose() {
		reset();
		onClose();
	}

	// ── CSV drag & drop ──────────────────────────────────────────────────────

	function onDragOver(e: DragEvent) {
		e.preventDefault();
		dragOver = true;
	}

	function onDragLeave() {
		dragOver = false;
	}

	function onDrop(e: DragEvent) {
		e.preventDefault();
		dragOver = false;
		const f = e.dataTransfer?.files[0];
		if (f) csvFile = f;
	}

	function onFileChange(e: Event) {
		const target = e.target as HTMLInputElement;
		csvFile = target.files?.[0] ?? null;
	}

	// ── Template download ────────────────────────────────────────────────────

	async function downloadTemplate() {
		try {
			const res = await productsApi.downloadTemplate();
			if (!res.ok) return;
			const blob = await res.blob();
			const url = URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			a.download = 'product_import_template.csv';
			a.click();
			URL.revokeObjectURL(url);
		} catch {
			// silently ignore
		}
	}

	// ── JSON validation ──────────────────────────────────────────────────────

	function validateJson() {
		if (!jsonText.trim()) {
			jsonError = '';
			return;
		}
		try {
			const parsed = JSON.parse(jsonText);
			if (!Array.isArray(parsed)) {
				jsonError = $t('products.jsonExpectsArray');
			} else {
				jsonError = '';
			}
		} catch (e: unknown) {
			jsonError = e instanceof Error ? e.message : $t('products.jsonInvalid');
		}
	}

	// ── Import actions ───────────────────────────────────────────────────────

	async function importCSV() {
		if (!csvFile) return;
		phase = 'loading';
		errorMsg = '';
		try {
			const res = await productsApi.importCSV(csvFile);
			result = res.data;
			phase = 'result';
			if (result.succeeded > 0) onImported();
		} catch (e: unknown) {
			errorMsg = e instanceof Error ? e.message : $t('products.importFailed');
			phase = 'idle';
		}
	}

	async function importJSON() {
		validateJson();
		if (jsonError) return;
		let products: unknown[];
		try {
			products = JSON.parse(jsonText);
		} catch {
			jsonError = $t('products.jsonInvalid');
			return;
		}
		phase = 'loading';
		errorMsg = '';
		try {
			const res = await productsApi.bulk({ products: products as never });
			result = res.data;
			phase = 'result';
			if (result.succeeded > 0) onImported();
		} catch (e: unknown) {
			errorMsg = e instanceof Error ? e.message : $t('products.importFailed');
			phase = 'idle';
		}
	}

	function toggleExpand(index: number) {
		expanded[index] = !expanded[index];
	}

	const title = $derived(phase === 'result' ? $t('products.importResult') : $t('products.importTitle'));
</script>

{#if open}
	<div
		class="fixed inset-0 z-40 flex items-center justify-center p-4"
		transition:fade={{ duration: 150 }}
	>
		<!-- Backdrop -->
		<!-- svelte-ignore a11y_no_static_element_interactions -->
		<div
			class="absolute inset-0 bg-black/40"
			onclick={handleClose}
			onkeydown={(e) => e.key === 'Escape' && handleClose()}
			role="button"
			tabindex="-1"
			aria-label={$t('modal.closeLabel')}
		></div>

		<!-- Panel -->
		<div
			class="relative z-50 bg-white rounded-xl shadow-2xl w-full max-w-2xl flex flex-col"
			style="max-height: 90vh;"
			transition:scale={{ duration: 150, start: 0.97 }}
		>
			<!-- Header -->
			<div class="flex items-center justify-between px-6 py-4 border-b border-gray-200 shrink-0">
				<h2 class="text-lg font-semibold text-gray-900">{title}</h2>
				<button
					onclick={handleClose}
					class="text-gray-400 hover:text-gray-600 transition-colors text-2xl leading-none w-8 h-8 flex items-center justify-center rounded hover:bg-gray-100"
					aria-label={$t('common.close')}
				>×</button>
			</div>

			<!-- Body -->
			<div class="flex-1 overflow-y-auto">
				{#if phase === 'loading'}
					<!-- Loading state -->
					<div class="flex flex-col items-center justify-center py-20 gap-4">
						<div class="relative w-12 h-12">
							<div class="absolute inset-0 rounded-full border-4 border-gray-100"></div>
							<div class="absolute inset-0 rounded-full border-4 border-t-primary-600 animate-spin"></div>
						</div>
						<p class="text-sm text-gray-500 font-medium">{$t('products.importProducts')}</p>
					</div>

				{:else if phase === 'result' && result}
					<!-- Result state -->
					<div class="p-6 space-y-5">
						<!-- Summary bar -->
						<div class="rounded-lg overflow-hidden border border-gray-200">
							<div
								class="px-5 py-4 flex items-center gap-4"
								style="background: {result.failed === 0 ? '#f0fdf4' : '#fff7ed'};"
							>
								<div
									class="w-10 h-10 rounded-full flex items-center justify-center shrink-0"
									style="background: {result.failed === 0 ? '#dcfce7' : '#ffedd5'};"
								>
									{#if result.failed === 0}
										<svg class="w-5 h-5" style="color:#16a34a" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2.5" d="M5 13l4 4L19 7"/>
										</svg>
									{:else}
										<svg class="w-5 h-5" style="color:#ea580c" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v4m0 4h.01M10.29 3.86L1.82 18a2 2 0 001.71 3h16.94a2 2 0 001.71-3L13.71 3.86a2 2 0 00-3.42 0z"/>
										</svg>
									{/if}
								</div>
								<div>
									<p class="font-semibold text-gray-900">
										{$t('products.importSucceeded', { values: { succeeded: result.succeeded, total: result.total } })}
									</p>
									{#if result.failed > 0}
										<p class="text-sm text-orange-700 mt-0.5">{$t('products.importFailedCount', { values: { failed: result.failed } })}</p>
									{/if}
								</div>
							</div>

							<!-- Progress bar -->
							<div class="h-1.5 bg-gray-100">
								<div
									class="h-full transition-all duration-700 ease-out"
									style="width: {result.total > 0 ? (result.succeeded / result.total) * 100 : 0}%; background: #16a34a;"
								></div>
							</div>
						</div>

						<!-- Failed items -->
						{#if result.failed > 0}
							<div class="space-y-2">
								<p class="text-xs font-semibold text-gray-500 uppercase tracking-wider">{$t('products.importErrorDetails')}</p>
								{#each result.results.filter(r => !r.success) as row}
									<div class="border border-red-100 rounded-lg overflow-hidden">
										<button
											class="w-full flex items-center justify-between px-4 py-3 text-sm bg-red-50 hover:bg-red-100 transition-colors text-left"
											onclick={() => toggleExpand(row.index)}
										>
											<span class="flex items-center gap-2 font-medium text-red-800">
												<svg class="w-4 h-4 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
													<circle cx="12" cy="12" r="10" stroke-width="2"/>
													<path stroke-linecap="round" stroke-width="2" d="M12 8v4m0 4h.01"/>
												</svg>
												{$t('products.importRow', { values: { row: row.index + 1 } })}{row.sku ? ` · ${row.sku}` : ''}
											</span>
											<svg
												class="w-4 h-4 text-red-500 transition-transform"
												class:rotate-180={expanded[row.index]}
												fill="none" stroke="currentColor" viewBox="0 0 24 24"
											>
												<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7"/>
											</svg>
										</button>
										{#if expanded[row.index]}
											<ul class="px-4 py-3 space-y-1 bg-white" transition:fade={{ duration: 100 }}>
												{#each row.errors ?? [] as err}
													<li class="text-xs text-red-700 flex gap-2">
														<span class="text-red-400 shrink-0">–</span>
														<span>{err}</span>
													</li>
												{/each}
											</ul>
										{/if}
									</div>
								{/each}
							</div>
						{/if}
					</div>

				{:else}
					<!-- Idle state: tabs -->
					<div>
						<!-- Tab bar -->
						<div class="flex border-b border-gray-200 px-6">
							{#each (['csv', 'json'] as const) as tab}
								<button
									class="px-4 py-3 text-sm font-medium transition-colors relative"
									class:text-primary-600={activeTab === tab}
									class:text-gray-500={activeTab !== tab}
									class:hover:text-gray-700={activeTab !== tab}
									onclick={() => { activeTab = tab; errorMsg = ''; }}
								>
									{tab === 'csv' ? $t('products.csvImport') : $t('products.jsonImport')}
									{#if activeTab === tab}
										<span class="absolute bottom-0 left-0 right-0 h-0.5 bg-primary-600 rounded-t"></span>
									{/if}
								</button>
							{/each}
						</div>

						<!-- Tab content -->
						<div class="p-6">
							{#if activeTab === 'csv'}
								<div class="space-y-4">
									<p class="text-sm text-gray-600">
										{$t('products.csvDescription')}
									</p>

									<!-- Template download -->
									<button
										onclick={downloadTemplate}
										class="inline-flex items-center gap-1.5 text-sm text-primary-600 hover:text-primary-700 font-medium transition-colors"
									>
										<svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
											<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 10v6m0 0l-3-3m3 3l3-3M3 17V7a2 2 0 012-2h6l2 2h6a2 2 0 012 2v8a2 2 0 01-2 2H5a2 2 0 01-2-2z"/>
										</svg>
										{$t('products.csvDownloadTemplate')}
									</button>

									<!-- Drop zone -->
									<!-- svelte-ignore a11y_no_static_element_interactions -->
									<div
										class="relative rounded-xl border-2 border-dashed transition-all duration-200 cursor-pointer"
										class:border-primary-400={dragOver}
										class:bg-primary-50={dragOver}
										class:border-gray-300={!dragOver}
										class:bg-gray-50={!dragOver && !csvFile}
										class:bg-white={!dragOver && !!csvFile}
										ondragover={onDragOver}
										ondragleave={onDragLeave}
										ondrop={onDrop}
										onclick={() => fileInput.click()}
										onkeydown={(e) => e.key === 'Enter' && fileInput.click()}
										role="button"
										tabindex="0"
										aria-label={$t('products.csvSelectFile')}
									>
										<input
											bind:this={fileInput}
											type="file"
											accept=".csv,text/csv"
											class="sr-only"
											onchange={onFileChange}
										/>

										<div class="flex flex-col items-center justify-center py-10 px-6 text-center">
											{#if csvFile}
												<div class="w-12 h-12 rounded-full bg-primary-100 flex items-center justify-center mb-3">
													<svg class="w-6 h-6 text-primary-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
														<path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"/>
													</svg>
												</div>
												<p class="font-medium text-gray-900 text-sm">{csvFile.name}</p>
												<p class="text-xs text-gray-400 mt-1">{(csvFile.size / 1024).toFixed(1)} KB · {$t('products.csvClickToChange')}</p>
											{:else}
												<div class="w-12 h-12 rounded-full bg-gray-100 flex items-center justify-center mb-3 transition-colors" class:bg-primary-100={dragOver}>
													<svg class="w-6 h-6 transition-colors" class:text-primary-500={dragOver} class:text-gray-400={!dragOver} fill="none" stroke="currentColor" viewBox="0 0 24 24">
														<path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12"/>
													</svg>
												</div>
												<p class="font-medium text-gray-700 text-sm">{$t('products.csvDropHere')}</p>
												<p class="text-xs text-gray-400 mt-1">{$t('products.csvOrClick')}</p>
											{/if}
										</div>
									</div>

									{#if errorMsg}
										<p class="text-sm text-red-600 flex items-center gap-1.5">
											<svg class="w-4 h-4 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
												<circle cx="12" cy="12" r="10" stroke-width="2"/>
												<path stroke-linecap="round" stroke-width="2" d="M12 8v4m0 4h.01"/>
											</svg>
											{errorMsg}
										</p>
									{/if}
								</div>

							{:else}
								<!-- JSON tab -->
								<div class="space-y-4">
									<p class="text-sm text-gray-600">
										{@html $t('products.jsonDescription', { values: { schema: '<code class="text-xs bg-gray-100 px-1.5 py-0.5 rounded font-mono">CreateProductRequest</code>', variants: '<code class="text-xs bg-gray-100 px-1.5 py-0.5 rounded font-mono">variants</code>' } })}
									</p>

									<div class="space-y-1.5">
										<textarea
											class="input w-full font-mono text-xs resize-none"
											class:border-red-400={!!jsonError}
											rows="14"
											placeholder={'[\n  {\n    "sku": "SHIRT-001",\n    "currency": "EUR",\n    "price_gross": 2499,\n    "translations": [{ "locale": "de", "name": "T-Shirt", "slug": "t-shirt" }]\n  }\n]'}
											bind:value={jsonText}
											oninput={validateJson}
											spellcheck="false"
										></textarea>
										{#if jsonError}
											<p class="text-xs text-red-600 font-mono">{jsonError}</p>
										{/if}
									</div>

									{#if errorMsg}
										<p class="text-sm text-red-600 flex items-center gap-1.5">
											<svg class="w-4 h-4 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24">
												<circle cx="12" cy="12" r="10" stroke-width="2"/>
												<path stroke-linecap="round" stroke-width="2" d="M12 8v4m0 4h.01"/>
											</svg>
											{errorMsg}
										</p>
									{/if}
								</div>
							{/if}
						</div>
					</div>
				{/if}
			</div>

			<!-- Footer -->
			<div class="px-6 py-4 border-t border-gray-200 flex justify-end gap-3 shrink-0">
				{#if phase === 'result'}
					<button class="btn btn-secondary" onclick={reset}>{$t('common.furtherImport')}</button>
					<button class="btn btn-primary" onclick={handleClose}>{$t('common.close')}</button>
				{:else if phase === 'idle'}
					<button class="btn btn-secondary" onclick={handleClose}>{$t('common.cancel')}</button>
					{#if activeTab === 'csv'}
						<button
							class="btn btn-primary"
							disabled={!csvFile}
							onclick={importCSV}
						>
							{$t('common.import')}
						</button>
					{:else}
						<button
							class="btn btn-primary"
							disabled={!jsonText.trim() || !!jsonError}
							onclick={importJSON}
						>
							{$t('common.import')}
						</button>
					{/if}
				{/if}
			</div>
		</div>
	</div>
{/if}
