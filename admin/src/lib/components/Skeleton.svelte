<script lang="ts">
	interface Props {
		class?: string;
		lines?: number;
		height?: string;
	}
	let { class: className = '', lines = 1, height = 'h-4' }: Props = $props();
</script>

{#each Array(lines) as _, i}
	<div class="skeleton {height} {className}" style:width={i === lines - 1 && lines > 1 ? '75%' : '100%'}></div>
{/each}
