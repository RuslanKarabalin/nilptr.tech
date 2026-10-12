<script lang="ts">
	let { error, success }: { error?: string | null; success?: string | null } = $props();
</script>

{#if error}
	<p class="message error" role="alert">{error}</p>
{:else if success}
	<p class="message success" role="status">{success}</p>
{/if}

<style>
	.message {
		display: flex;
		align-items: center;
		gap: 0.6rem;
		padding: 0.7rem 1rem;
		border-radius: var(--radius-sm);
		font-size: 0.9375rem;
		font-weight: 500;
	}

	.message::before {
		content: '';
		width: 0.5rem;
		height: 0.5rem;
		flex: none;
		border-radius: 50%;
		background: currentColor;
	}

	.error {
		background: var(--danger-soft);
		color: var(--danger);
	}

	.success {
		background: var(--success-soft);
		color: var(--success);
	}
</style>
