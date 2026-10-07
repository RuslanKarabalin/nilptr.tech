<script lang="ts">
	import { resolve } from '$app/paths';
	import type { NavLink as Link } from '$lib/types';
	import NavLink from './NavLink.svelte';
	import ThemeToggle from './ThemeToggle.svelte';

	let { links }: { links: Link[] } = $props();
</script>

<header class="site-header">
	<nav class="container">
		<ul>
			<li>
				<a href={resolve('/')}>Home</a>
			</li>
		</ul>
		<ul>
			{#each links as link, i (i)}
				<li>
					<NavLink {link} />
				</li>
			{/each}
		</ul>
		<ul>
			<li>
				<ThemeToggle />
			</li>
		</ul>
	</nav>
</header>

<style>
	.site-header {
		position: sticky;
		top: 0;
		z-index: 1;
		border-bottom: 1px solid var(--pico-muted-border-color);
		background: var(--pico-background-color);
	}
</style>
