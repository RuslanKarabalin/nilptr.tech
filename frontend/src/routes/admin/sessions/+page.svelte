<script lang="ts">
	import { enhance } from '$app/forms';
	import FormMessage from '$lib/components/admin/FormMessage.svelte';
	import { formatDateTime } from '$lib/format';

	let { data, form } = $props();
</script>

<svelte:head>
	<title>Sessions - nilptr admin</title>
</svelte:head>

<h1>Sessions</h1>
<p>
	<small>
		Every device you logged in from. Ending a session logs that device out immediately.
	</small>
</p>

<FormMessage error={form?.error} success={form && 'success' in form ? form.success : null} />

{#if data.sessions.length === 0}
	<p>No active sessions.</p>
{:else}
	<div class="overflow-auto">
		<table>
			<thead>
				<tr>
					<th>Device</th>
					<th>IP</th>
					<th>Logged in</th>
					<th>Last used</th>
					<th></th>
				</tr>
			</thead>
			<tbody>
				{#each data.sessions as session (session.id)}
					<tr class:current={session.current}>
						<td>
							{#if session.current}<mark>current</mark>{/if}
							<small class="ua">{session.user_agent || 'unknown'}</small>
						</td>
						<td><small>{session.ip}</small></td>
						<td><small>{formatDateTime(session.created_at)}</small></td>
						<td><small>{formatDateTime(session.last_used_at)}</small></td>
						<td>
							<form
								method="POST"
								action="?/end"
								use:enhance={({ cancel }) => {
									if (
										session.current &&
										!confirm('End the current session? You will be logged out.')
									) {
										cancel();
									}
								}}
							>
								<input type="hidden" name="id" value={session.id} />
								<input type="hidden" name="current" value={String(session.current)} />
								<button type="submit" class="outline secondary small">End session</button>
							</form>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
{/if}

<style>
	td form {
		margin: 0;
	}

	.ua {
		display: block;
		overflow-wrap: anywhere;
	}

	.small {
		margin: 0;
		padding: 0.125rem 0.5rem;
		font-size: 0.875rem;
		white-space: nowrap;
	}
</style>
