<script lang="ts">
	let { data } = $props();

	let max = $derived(Math.max(1, ...data.items.map((item) => item.count)));
</script>

<svelte:head>
	<title>Stats - nilptr admin</title>
</svelte:head>

<h1>Stats</h1>

<form method="GET" class="range">
	<label>
		From
		<input type="date" name="from" value={data.from} required />
	</label>
	<label>
		To
		<input type="date" name="to" value={data.to} required />
	</label>
	<button type="submit">Show</button>
</form>

<p>
	<strong>{data.total}</strong> views from {data.from} to {data.to} (UTC, inclusive).
</p>

{#if data.items.length === 0}
	<p>No views in this period.</p>
{:else}
	<div class="overflow-auto">
		<table>
			<thead>
				<tr><th>Path</th><th class="count">Views</th><th class="bar-cell"></th></tr>
			</thead>
			<tbody>
				{#each data.items as item (item.path)}
					<tr>
						<td><code>{item.path}</code></td>
						<td class="count">{item.count}</td>
						<td class="bar-cell">
							<span class="bar" style:width="{(item.count / max) * 100}%"></span>
						</td>
					</tr>
				{/each}
			</tbody>
		</table>
	</div>
{/if}

<style>
	.range {
		display: flex;
		flex-wrap: wrap;
		align-items: flex-end;
		gap: 0.75rem 1rem;
		margin-bottom: 1.5rem;
	}

	.range label {
		flex: 1 1 10rem;
		margin: 0;
	}

	.range input {
		margin: 0.35rem 0 0;
	}

	.range button {
		flex: 0 0 auto;
		width: auto;
		min-width: 7rem;
		margin: 0;
	}

	@media (max-width: 480px) {
		.range button {
			flex-basis: 100%;
		}
	}

	.count {
		text-align: right;
		white-space: nowrap;
	}

	.bar-cell {
		width: 40%;
	}

	.bar {
		display: block;
		height: 0.5rem;
		min-width: 2px;
		border-radius: 0.25rem;
		background: var(--pico-primary-background);
	}
</style>
