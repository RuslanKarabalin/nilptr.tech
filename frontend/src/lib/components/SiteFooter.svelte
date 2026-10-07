<script lang="ts">
	import { SITE_NAME } from '$lib/nav';
	import type { NavLink as Link } from '$lib/types';
	import NavLink from './NavLink.svelte';

	let { links }: { links: Link[] } = $props();
</script>

<footer class="site-footer">
	<nav class="container">
		<ul>
			<li><small>{SITE_NAME}</small></li>
		</ul>
		<ul>
			{#each links as link, i (i)}
				<li>
					<small><NavLink {link} /></small>
				</li>
			{/each}
		</ul>
	</nav>
</footer>

<style>
	.site-footer {
		border-top: 1px solid var(--pico-muted-border-color);
	}
</style>
