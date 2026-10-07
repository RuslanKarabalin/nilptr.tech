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
		padding: 0.5rem 0.75rem;
		border-radius: var(--pico-border-radius);
		border: 1px solid currentColor;
	}

	.error {
		color: var(--pico-del-color);
	}

	.success {
		color: var(--pico-ins-color);
	}
</style>
