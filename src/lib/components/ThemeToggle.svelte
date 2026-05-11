<script lang="ts">
	import { onMount } from 'svelte';

	type Theme = 'light' | 'dark';

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

<style>
	.theme-toggle {
		background: transparent;
		border: none;
		color: inherit;
	}
</style>
