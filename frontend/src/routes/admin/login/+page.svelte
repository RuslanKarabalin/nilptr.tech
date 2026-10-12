<script lang="ts">
	import { enhance } from '$app/forms';
	import { resolve } from '$app/paths';
	import Brand from '$lib/components/Brand.svelte';
	import Icon from '$lib/components/Icon.svelte';
	import FormMessage from '$lib/components/admin/FormMessage.svelte';

	let { form } = $props();
	let busy = $state(false);
</script>

<svelte:head>
	<title>Log in - nilptr admin</title>
</svelte:head>

<div class="login">
	<div class="brand"><Brand admin /></div>

	<section class="card">
		<h1>Welcome back</h1>
		<p class="hint">Sign in to manage posts, pages and comments.</p>
		<form
			method="POST"
			use:enhance={() => {
				busy = true;
				return async ({ update }) => {
					await update();
					busy = false;
				};
			}}
		>
			<FormMessage error={form?.error} />
			<label>
				Login
				<input
					name="login"
					type="text"
					autocomplete="username"
					required
					value={form?.login ?? ''}
				/>
			</label>
			<label>
				Password
				<input name="password" type="password" autocomplete="current-password" required />
			</label>
			<button type="submit" aria-busy={busy} disabled={busy}>Log in</button>
		</form>
	</section>

	<a class="back" href={resolve('/')}><Icon name="arrow-left" /> Back to the site</a>
</div>

<style>
	.login {
		display: grid;
		gap: 1.5rem;
		justify-items: center;
		width: 100%;
		max-width: 25rem;
	}

	.card {
		width: 100%;
		padding: 2rem;
		box-sizing: border-box;
		border-radius: var(--radius-lg);
		box-shadow: var(--shadow-lg);
	}

	h1 {
		margin-bottom: 0.25rem;
		font-size: 1.5rem;
	}

	.hint {
		margin-bottom: 1.5rem;
		color: var(--muted);
		font-size: 0.9375rem;
	}

	button {
		width: 100%;
		margin-bottom: 0;
	}

	.back {
		display: inline-flex;
		align-items: center;
		gap: 0.4rem;
		color: var(--muted);
		font-size: 0.875rem;
		text-decoration: none;
	}

	.back:hover {
		color: var(--accent);
	}
</style>
