<script lang="ts">
	import Modal from './Modal.svelte';
	import { t } from 'svelte-i18n';

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
		title,
		message,
		confirmLabel,
		danger = false,
		onConfirm,
		onCancel
	}: Props = $props();
</script>

<Modal open={open} title={title ?? $t('common.confirm')} onClose={onCancel}>
	<p class="text-[var(--text-muted)]">{message}</p>
	{#snippet footer()}
		<button class="btn btn-secondary" onclick={onCancel}>{$t('common.cancel')}</button>
		<button
			class={danger ? 'btn btn-danger' : 'btn btn-primary'}
			onclick={onConfirm}
		>{confirmLabel ?? $t('common.confirm')}</button>
	{/snippet}
</Modal>
