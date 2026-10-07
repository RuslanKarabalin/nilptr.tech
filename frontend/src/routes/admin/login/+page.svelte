<script lang="ts">
	import { enhance } from '$app/forms';
	import FormMessage from '$lib/components/admin/FormMessage.svelte';

	let { form } = $props();
	let busy = $state(false);
</script>

<svelte:head>
	<title>Log in - nilptr admin</title>
</svelte:head>

<article class="login">
	<h1>Log in</h1>
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
			<input name="login" type="text" autocomplete="username" required value={form?.login ?? ''} />
		</label>
		<label>
			Password
			<input name="password" type="password" autocomplete="current-password" required />
		</label>
		<button type="submit" aria-busy={busy} disabled={busy}>Log in</button>
	</form>
</article>

<style>
	.login {
		max-width: 28rem;
		margin-inline: auto;
	}
</style>
