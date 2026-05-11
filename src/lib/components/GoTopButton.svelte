<script lang="ts">
	import { onMount } from 'svelte';

	let showGoTop = $state(false);

	onMount(() => {
		const updateGoTopVisibility = () => {
			showGoTop = window.scrollY > 128;
		};

		updateGoTopVisibility();
		window.addEventListener('scroll', updateGoTopVisibility, { passive: true });

		return () => {
			window.removeEventListener('scroll', updateGoTopVisibility);
		};
	});

	let goToTop = () => {
		window.scrollTo({ top: 0, behavior: 'smooth' });
	};
</script>

{#if showGoTop}
	<button type="button" class="go-top" onclick={goToTop} aria-label="Go to top" title="Go to top">
		^
	</button>
{/if}

<style>
	.go-top {
		position: fixed;
		right: 1rem;
		bottom: 2rem;
		z-index: 1;

		width: 3rem;
		height: 3rem;
		border-radius: 128px;

		display: grid;
		place-items: center;
	}
</style>
