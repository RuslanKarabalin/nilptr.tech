<script lang="ts">
	import favicon from "$lib/assets/favicon.svg";
	import "@picocss/pico/css/pico.min.css";

	let { children } = $props();
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
</svelte:head>

<div class="page">
	<header class="site-header">
		<nav class="container">
			<ul>
				<li>
					<a href="/" class="brand">nilptr.tech</a>
				</li>
			</ul>

			<ul>
				<li><a href="/">Home</a></li>
				<li><a href="/cv">CV</a></li>
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
						<a
							href="https://github.com/RuslanKarabalin"
							target="_blank"
							rel="noopener noreferrer"
						>
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

	.container {
		width: min(100% - 2rem, 48rem);
		margin-inline: auto;
	}

	.site-header {
		border-bottom: 1px solid var(--pico-muted-border-color);
	}

	.site-header nav {
		padding-block: 1rem;
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
		padding-block: 1rem;
	}

	.site-footer nav,
	.site-footer ul {
		margin: 0;
	}
</style>
