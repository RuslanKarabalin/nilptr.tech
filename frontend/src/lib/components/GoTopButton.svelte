<script lang="ts">
	import { onMount } from 'svelte';
	import Icon from './Icon.svelte';

	let visible = $state(false);

	onMount(() => {
		const update = () => {
			visible = window.scrollY > 400;
		};

		update();
		window.addEventListener('scroll', update, { passive: true });

		return () => {
			window.removeEventListener('scroll', update);
		};
	});

	function goToTop() {
		window.scrollTo({ top: 0, behavior: 'smooth' });
	}
</script>

<button
	type="button"
	class="go-top"
	class:visible
	onclick={goToTop}
	aria-label="Go to top"
	title="Go to top"
	tabindex={visible ? 0 : -1}
	aria-hidden={!visible}
>
	<Icon name="arrow-up" size="1.2rem" />
</button>

<style>
	.go-top {
		position: fixed;
		right: 1.25rem;
		bottom: 1.25rem;
		z-index: 10;
		display: grid;
		place-items: center;
		width: 2.75rem;
		height: 2.75rem;
		margin: 0;
		padding: 0;
		border: 1px solid var(--border);
		border-radius: 999px;
		background: var(--surface);
		color: var(--text);
		box-shadow: var(--shadow-lg);
		cursor: pointer;
		opacity: 0;
		transform: translateY(0.75rem);
		pointer-events: none;
		transition:
			opacity 0.2s var(--ease),
			transform 0.2s var(--ease),
			border-color 0.15s var(--ease),
			color 0.15s var(--ease);
	}

	.go-top.visible {
		opacity: 1;
		transform: none;
		pointer-events: auto;
	}

	.go-top:hover {
		border-color: var(--accent);
		color: var(--accent);
	}
</style>
