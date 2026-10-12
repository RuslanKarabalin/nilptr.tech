<script lang="ts">
	import { afterNavigate } from '$app/navigation';
	import { page } from '$app/state';
	import { isInternalUrl } from '$lib/format';
	import type { NavLink as Link } from '$lib/types';
	import Brand from './Brand.svelte';
	import Icon from './Icon.svelte';
	import NavLink from './NavLink.svelte';
	import ThemeToggle from './ThemeToggle.svelte';

	let { links }: { links: Link[] } = $props();

	// The mobile menu is a <details>, so it also opens without JavaScript.
	let menuOpen = $state(false);
	let menu = $state<HTMLDetailsElement>();

	afterNavigate(() => {
		menuOpen = false;
	});

	function isCurrent(link: Link): boolean {
		const path = page.url.pathname;
		return isInternalUrl(link.url) && (path === link.url || path.startsWith(link.url + '/'));
	}

	function onWindowClick(event: MouseEvent) {
		if (menuOpen && menu && !menu.contains(event.target as Node)) menuOpen = false;
	}

	function onWindowKeydown(event: KeyboardEvent) {
		if (menuOpen && event.key === 'Escape') {
			menuOpen = false;
			menu?.querySelector('summary')?.focus();
		}
	}
</script>

<svelte:window onclick={onWindowClick} onkeydown={onWindowKeydown} />

<header class="site-header">
	<div class="wrap bar">
		<Brand />

		{#if links.length}
			<nav class="desktop-nav" aria-label="Main">
				<ul>
					{#each links as link, i (i)}
						<li><NavLink {link} current={isCurrent(link)} /></li>
					{/each}
				</ul>
			</nav>
		{/if}

		<div class="tools">
			<ThemeToggle />
			{#if links.length}
				<details class="mobile-menu" bind:open={menuOpen} bind:this={menu}>
					<summary class="icon-button" aria-label="Menu" title="Menu">
						<Icon name="menu" />
					</summary>
					<nav class="mobile-panel card" aria-label="Main">
						<ul>
							{#each links as link, i (i)}
								<li><NavLink {link} current={isCurrent(link)} /></li>
							{/each}
						</ul>
					</nav>
				</details>
			{/if}
		</div>
	</div>
</header>

<style>
	.site-header {
		position: sticky;
		top: 0;
		z-index: 10;
		border-bottom: 1px solid var(--border);
		background: var(--header-bg);
		backdrop-filter: saturate(1.6) blur(14px);
		-webkit-backdrop-filter: saturate(1.6) blur(14px);
	}

	.bar {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		height: var(--header-height);
	}

	/* The brand pushes the rest to the right, with or without nav links. */
	.bar > :global(.brand) {
		margin-right: auto;
	}

	.desktop-nav {
		min-width: 0;
	}

	.tools {
		display: flex;
		align-items: center;
		gap: 0.25rem;
	}

	ul {
		display: flex;
		gap: 0.25rem;
		margin: 0;
		padding: 0;
		list-style: none;
	}

	li {
		margin: 0;
		padding: 0;
		list-style: none;
	}

	nav :global(a) {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		margin: 0;
		padding: 0.45rem 0.8rem;
		border-radius: 999px;
		color: var(--muted);
		font-size: 0.9375rem;
		font-weight: 500;
		text-decoration: none;
		white-space: nowrap;
		transition:
			background 0.15s var(--ease),
			color 0.15s var(--ease);
	}

	nav :global(a:hover) {
		background: var(--surface-2);
		color: var(--heading);
	}

	nav :global(a[aria-current='page']) {
		background: var(--accent-soft);
		color: var(--accent);
	}

	/* Mobile menu */

	.mobile-menu {
		display: none;
		position: relative;
		margin: 0;
		padding: 0;
		border: 0;
	}

	.mobile-menu summary {
		list-style: none;
		margin: 0;
		padding: 0;
		line-height: 1;
	}

	.mobile-menu summary::-webkit-details-marker,
	.mobile-menu summary::after {
		display: none;
	}

	.mobile-menu[open] summary {
		background: var(--surface-2);
		border-color: var(--border);
		color: var(--heading);
	}

	.mobile-panel {
		position: absolute;
		top: calc(100% + 0.6rem);
		right: 0;
		min-width: 13rem;
		padding: 0.4rem;
		box-shadow: var(--shadow-lg);
	}

	.mobile-panel ul {
		flex-direction: column;
		align-items: stretch;
		gap: 0.1rem;
	}

	.mobile-panel :global(a) {
		display: flex;
		padding: 0.65rem 0.85rem;
		border-radius: var(--radius-sm);
		font-size: 1rem;
	}

	@media (max-width: 640px) {
		.desktop-nav {
			display: none;
		}

		.mobile-menu {
			display: block;
		}
	}
</style>
