<script lang="ts">
	import { resolve } from '$app/paths';
	import { page } from '$app/state';
	import Icon from '$lib/components/Icon.svelte';
	import { SITE_NAME } from '$lib/nav';

	let notFound = $derived(page.status === 404);
	let title = $derived(notFound ? 'Page not found' : 'Something went wrong');
</script>

<svelte:head>
	<title>{page.status} {title} - {SITE_NAME}</title>
	<meta name="robots" content="noindex" />
</svelte:head>

<section class="wrap error-page">
	<p class="code">{page.status}</p>
	<h1>{title}</h1>
	<p class="hint">
		{#if notFound}
			This address points nowhere. Like a nil pointer.
		{:else if page.error?.message}
			{page.error.message}
		{:else}
			Please try again in a moment.
		{/if}
	</p>
	<a href={resolve('/')} role="button"><Icon name="arrow-left" /> Back to home</a>
</section>

<style>
	.error-page {
		display: grid;
		justify-items: center;
		padding-block: 4rem 2rem;
		text-align: center;
	}

	.code {
		margin: 0;
		font-family: var(--font-mono);
		font-size: clamp(4.5rem, 16vw, 8rem);
		font-weight: 700;
		line-height: 1;
		letter-spacing: -0.05em;
		background: linear-gradient(
			135deg,
			var(--accent),
			color-mix(in srgb, var(--accent) 35%, var(--heading))
		);
		-webkit-background-clip: text;
		background-clip: text;
		color: transparent;
	}

	h1 {
		margin: 1rem 0 0.5rem;
		font-size: 1.75rem;
	}

	.hint {
		max-width: 28rem;
		margin-bottom: 2rem;
		color: var(--muted);
	}

	[role='button'] {
		display: inline-flex;
		align-items: center;
		gap: 0.5rem;
	}
</style>
