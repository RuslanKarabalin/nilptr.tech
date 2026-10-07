<script lang="ts">
	import { resolve } from '$app/paths';
	import ThemeToggle from './ThemeToggle.svelte';
</script>

<header class="site-header">
	<nav class="container">
		<ul>
			<li>
				<a href={resolve('/')}>Home</a>
			</li>
		</ul>
		<ul>
			<li>
				<a href={resolve('/cv')}>CV</a>
			</li>
			<li>
				<a href="https://github.com/RuslanKarabalin" target="_blank" rel="noopener noreferrer">
					GitHub
				</a>
			</li>
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
