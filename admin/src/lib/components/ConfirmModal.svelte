<script lang="ts">
	import Modal from './Modal.svelte';

	interface Props {
		open: boolean;
		title?: string;
		message: string;
		confirmLabel?: string;
		danger?: boolean;
		onConfirm: () => void;
		onCancel: () => void;
	}
	let {
		open,
		title = 'Bestätigen',
		message,
		confirmLabel = 'Bestätigen',
		danger = false,
		onConfirm,
		onCancel
	}: Props = $props();
</script>

<Modal {open} {title} onClose={onCancel}>
	<p class="text-gray-700">{message}</p>
	{#snippet footer()}
		<button class="btn btn-secondary" onclick={onCancel}>Abbrechen</button>
		<button
			class={danger ? 'btn btn-danger' : 'btn btn-primary'}
			onclick={onConfirm}
		>{confirmLabel}</button>
	{/snippet}
</Modal>
