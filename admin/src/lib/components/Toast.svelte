<script lang="ts">
	import { notifications } from '$lib/stores/notifications';
	import { fly } from 'svelte/transition';
</script>

<div class="fixed bottom-4 right-4 z-50 flex flex-col gap-2 max-w-sm w-full">
	{#each $notifications as n (n.id)}
		<div
			transition:fly={{ x: 20, duration: 200 }}
			class="flex items-start gap-3 p-4 rounded-xl shadow-lg border text-sm"
			class:bg-green-50={n.type === 'success'}
			class:border-green-200={n.type === 'success'}
			class:bg-red-50={n.type === 'error'}
			class:border-red-200={n.type === 'error'}
			class:bg-blue-50={n.type === 'info'}
			class:border-blue-200={n.type === 'info'}
			class:bg-yellow-50={n.type === 'warning'}
			class:border-yellow-200={n.type === 'warning'}
		>
			<span class="text-lg leading-none mt-0.5">
				{#if n.type === 'success'}✓{:else if n.type === 'error'}✕{:else if n.type === 'warning'}⚠{:else}ℹ{/if}
			</span>
			<p class="flex-1"
				class:text-green-800={n.type === 'success'}
				class:text-red-800={n.type === 'error'}
				class:text-blue-800={n.type === 'info'}
				class:text-yellow-800={n.type === 'warning'}
			>{n.message}</p>
			<button
				onclick={() => notifications.remove(n.id)}
				class="text-gray-400 hover:text-gray-600"
			>×</button>
		</div>
	{/each}
</div>
