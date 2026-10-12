<script lang="ts">
	import { SITE_DESCRIPTION, SITE_NAME } from '$lib/nav';
	import type { NavLink as Link } from '$lib/types';
	import Brand from './Brand.svelte';
	import NavLink from './NavLink.svelte';

	let { links }: { links: Link[] } = $props();

	const year = new Date().getUTCFullYear();
</script>

<footer class="site-footer">
	<div class="wrap inner">
		<div class="about">
			<Brand />
			<p>{SITE_DESCRIPTION}</p>
		</div>
		{#if links.length}
			<nav aria-label="Footer">
				{#each links as link, i (i)}
					<NavLink {link} />
				{/each}
			</nav>
		{/if}
	</div>
	<div class="wrap legal">
		<small>&copy; {year} {SITE_NAME}</small>
	</div>
</footer>

<style>
	.site-footer {
		margin-top: 4rem;
		border-top: 1px solid var(--border);
		background: color-mix(in srgb, var(--surface) 60%, transparent);
	}

	.inner {
		display: flex;
		flex-wrap: wrap;
		justify-content: space-between;
		align-items: flex-start;
		gap: 1.5rem;
		padding-block: 2.5rem 1.5rem;
	}

	.about p {
		margin: 0.6rem 0 0;
		color: var(--muted);
		font-size: 0.9375rem;
	}

	nav {
		display: flex;
		flex-wrap: wrap;
		gap: 0.5rem;
	}

	nav :global(a) {
		display: inline-flex;
		margin: 0;
		align-items: center;
		gap: 0.4rem;
		padding: 0.45rem 0.85rem;
		border: 1px solid var(--border);
		border-radius: 999px;
		background: var(--surface);
		color: var(--text);
		font-size: 0.875rem;
		font-weight: 500;
		text-decoration: none;
		transition:
			border-color 0.15s var(--ease),
			color 0.15s var(--ease);
	}

	nav :global(a:hover) {
		border-color: var(--accent);
		color: var(--accent);
	}

	.legal {
		padding-block: 1rem 1.5rem;
		border-top: 1px solid var(--border);
		color: var(--muted);
	}
</style>
