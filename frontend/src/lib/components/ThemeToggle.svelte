<script lang="ts">
	import Icon from './Icon.svelte';

	// The current theme lives on <html data-theme>, set by app.html before the
	// first paint. The icon is chosen in CSS, so SSR and hydration always match.
	function toggleTheme() {
		const root = document.documentElement;
		const next = root.getAttribute('data-theme') === 'dark' ? 'light' : 'dark';
		root.setAttribute('data-theme', next);
		try {
			localStorage.setItem('theme', next);
		} catch {
			// localStorage may be unavailable
		}
	}
</script>

<button
	type="button"
	class="icon-button theme-toggle"
	onclick={toggleTheme}
	aria-label="Toggle dark theme"
	title="Toggle theme"
>
	<span class="sun"><Icon name="sun" /></span>
	<span class="moon"><Icon name="moon" /></span>
</button>

<style>
	.theme-toggle span {
		display: grid;
		place-items: center;
	}

	.moon,
	:global([data-theme='dark']) .sun {
		display: none !important;
	}

	:global([data-theme='dark']) .moon {
		display: grid !important;
	}
</style>
