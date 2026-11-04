<script lang="ts">
  import { errorMessage, pb } from '../lib/pb';
  import { session } from '../lib/session.svelte';
  import Brand from './Brand.svelte';

  let name = $state('');
  let email = $state('');
  let password = $state('');
  let code = $state('');
  let busy = $state(false);
  let error = $state('');

  // phones only install the app over HTTPS
  const insecure = !window.isSecureContext;

  async function submit(e: SubmitEvent) {
    e.preventDefault();
    busy = true;
    error = '';
    try {
      await pb.send('/api/app/setup', { method: 'POST', body: { name, email, password, code } });
      await session.login(email, password);
    } catch (err) {
      error = errorMessage(err, 'Could not save. Please try again.');
    } finally {
      busy = false;
    }
  }
</script>

<main class="mx-auto flex min-h-dvh max-w-sm flex-col justify-center px-6 py-10">
  <Brand title="Welcome" subtitle="Create your account to get started." />
  <form class="flex flex-col gap-4" onsubmit={submit}>
    <label>
      Setup code
      <input
        autocomplete="off"
        autocapitalize="characters"
        spellcheck="false"
        bind:value={code}
        required
        placeholder="ABCD-EF23"
      />
    </label>
    <p class="-mt-2 text-sm text-zinc-500 dark:text-zinc-400">
      The installer showed it. It's also in the app's log: <code>docker compose logs app</code>, or
      the container's log in your NAS's Docker app.
    </p>
    <label>Name <input autocomplete="name" bind:value={name} required maxlength="100" /></label>
    <label>Email <input type="email" autocomplete="username" bind:value={email} required /></label>
    <label>
      Password
      <input
        type="password"
        autocomplete="new-password"
        bind:value={password}
        required
        minlength="8"
        placeholder="At least 8 characters"
      />
    </label>
    <p class="text-sm text-zinc-500 dark:text-zinc-400">
      You'll be the owner. The same email and password also open the admin page at <code>/_/</code>.
    </p>
    {#if error}<p class="text-sm text-red-600 dark:text-red-400" role="alert">{error}</p>{/if}
    <button class="btn-primary" disabled={busy}>{busy ? 'Setting up…' : 'Create account'}</button>
    {#if insecure}
      <p
        class="rounded-xl bg-amber-100 px-4 py-3 text-sm text-amber-900 dark:bg-amber-900/40 dark:text-amber-100"
      >
        This address isn't HTTPS. The app works in a browser, but phones need HTTPS to install it.
      </p>
    {/if}
  </form>
</main>
