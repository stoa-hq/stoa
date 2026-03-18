<script lang="ts">
	interface Props {
		class?: string;
		iconOnly?: boolean;
	}
	let { class: className = '', iconOnly = false }: Props = $props();
</script>

{#if iconOnly}
	<svg viewBox="0 0 35 36" width="24" height="24" class={className} xmlns="http://www.w3.org/2000/svg">
		<defs>
			<linearGradient id="pillar-grad-icon" x1="0" y1="0" x2="0.4" y2="1">
				<stop offset="0%" stop-color="#c026d3"/>
				<stop offset="100%" stop-color="#86198f"/>
			</linearGradient>
		</defs>
		<polygon points="2,2 11,2 15.5,34 6.5,34" fill="url(#pillar-grad-icon)"/>
		<polygon points="19,2 28,2 32.5,34 23.5,34" fill="url(#pillar-grad-icon)"/>
	</svg>
{:else}
	<svg viewBox="0 0 156 36" height="28" class={className} xmlns="http://www.w3.org/2000/svg">
		<defs>
			<linearGradient id="pillar-grad-full" x1="0" y1="0" x2="0.4" y2="1">
				<stop offset="0%" stop-color="#c026d3"/>
				<stop offset="100%" stop-color="#86198f"/>
			</linearGradient>
			<linearGradient id="text-grad-full" x1="0" y1="0" x2="1" y2="0">
				<stop offset="0%" stop-color="#a21caf"/>
				<stop offset="100%" stop-color="#86198f"/>
			</linearGradient>
		</defs>
		<polygon points="2,2 11,2 15.5,34 6.5,34" fill="url(#pillar-grad-full)"/>
		<polygon points="19,2 28,2 32.5,34 23.5,34" fill="url(#pillar-grad-full)"/>
		<text
			x="43"
			y="27"
			font-family="'Rajdhani', 'Exo 2', 'Arial Narrow', sans-serif"
			font-size="24"
			font-weight="600"
			fill="url(#text-grad-full)"
			letter-spacing="2"
		>stoa</text>
	</svg>
{/if}
