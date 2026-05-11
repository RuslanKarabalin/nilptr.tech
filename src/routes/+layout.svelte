<script lang="ts">
	import { resolve } from '$app/paths';
	import { onMount } from 'svelte';

	type Theme = 'light' | 'dark';

	let { children } = $props();
	let theme = $state<Theme>('light');

	onMount(() => {
		const current = document.documentElement.getAttribute('data-theme');
		theme = current === 'dark' ? 'dark' : 'light';
	});

	let toggleTheme = () => {
		theme = theme === 'dark' ? 'light' : 'dark';
		document.documentElement.setAttribute('data-theme', theme);
		try {
			localStorage.setItem('theme', theme);
		} catch {
			// localStorage may be unavailable
		}
	};
</script>

<div class="page">
	<header class="site-header">
		<nav class="container">
			<ul>
				<li>
					<a href={resolve('/')} class="brand">nilptr.tech</a>
				</li>
			</ul>

			<ul>
				<li><a href={resolve('/cv')}>CV</a></li>
			</ul>
			<ul>
				<li>
					<button
						type="button"
						class="theme-toggle"
						onclick={toggleTheme}
						aria-label="Toggle theme"
						aria-pressed={theme === 'dark'}
						title={theme === 'dark' ? 'Switch to light theme' : 'Switch to dark theme'}
					>
						{theme === 'dark' ? 'dark' : 'light'}
					</button>
				</li>
			</ul>
		</nav>
	</header>

	<main class="container">
		{@render children()}
	</main>

	<footer class="site-footer">
		<div class="container footer-inner">
			<small>&copy; {new Date().getFullYear()} nilptr.tech</small>

			<nav>
				<ul>
					<li>
						<a href="https://github.com/RuslanKarabalin" target="_blank" rel="noopener noreferrer">
							GitHub
						</a>
					</li>
				</ul>
			</nav>
		</div>
	</footer>
</div>

<style>
	:global(html, body) {
		margin: 0;
		padding: 0;
	}

	.page {
		min-height: 100vh;
		display: flex;
		flex-direction: column;
	}

	.site-header {
		border-bottom: 1px solid var(--pico-muted-border-color);
	}

	.brand {
		font-weight: 700;
		text-decoration: none;
	}

	main {
		flex: 1;
		padding-block: 2rem;
	}

	.site-footer {
		border-top: 1px solid var(--pico-muted-border-color);
	}

	.footer-inner {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
	}

	.theme-toggle {
		background: transparent;
		border: none;
		color: inherit;
	}
</style>
