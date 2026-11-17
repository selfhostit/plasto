<script lang="ts">
  import { errorMessage, pb } from '../lib/pb';
  import { session } from '../lib/session.svelte';

  const user = $derived(session.user!);

  let open = $state(false);
  let current = $state('');
  let password = $state('');
  let again = $state('');
  let busy = $state(false);
  let error = $state('');
  let done = $state(false);

  async function change(e: SubmitEvent) {
    e.preventDefault();
    error = '';
    if (password !== again) {
      error = "The new passwords don't match.";
      return;
    }
    busy = true;
    try {
      await pb.collection('users').update(user.id, {
        oldPassword: current,
        password,
        passwordConfirm: again,
      });
    } catch (err) {
      const data = (err as { response?: { data?: Record<string, unknown> } }).response?.data;
      error = data?.oldPassword
        ? 'Your current password is wrong.'
        : data?.password
          ? 'The new password needs at least 8 characters.'
          : errorMessage(err);
      busy = false;
      return;
    }
    // a new password signs out every device, this one too: sign in again with it
    try {
      await session.login(user.email, password);
      current = password = again = '';
      open = false;
      done = true;
    } catch {
      pb.authStore.clear();
    } finally {
      busy = false;
    }
  }
</script>

<section class="card flex flex-col gap-3 text-sm text-zinc-600 dark:text-zinc-400">
  <p>Signed in as {user.email}{session.isOwner ? ' (owner)' : ''}.</p>
  <div class="flex flex-wrap gap-2">
    {#if session.isOwner}
      <a class="btn-secondary px-3 py-2 text-sm" href="#/people">People</a>
    {/if}
    <button
      class="btn-secondary px-3 py-2 text-sm"
      onclick={() => ((open = !open), (done = false), (error = ''))}
      aria-expanded={open}>Change password</button
    >
  </div>
  {#if done}<p role="status">Your password is changed. Other devices need to sign in again.</p>{/if}
  {#if open}
    <form class="flex flex-col gap-4 text-zinc-900 dark:text-zinc-100" onsubmit={change}>
      <label>
        Current password
        <input type="password" autocomplete="current-password" bind:value={current} required />
      </label>
      <label>
        New password
        <input
          type="password"
          autocomplete="new-password"
          bind:value={password}
          required
          minlength="8"
          placeholder="At least 8 characters"
        />
      </label>
      <label>
        New password again
        <input
          type="password"
          autocomplete="new-password"
          bind:value={again}
          required
          minlength="8"
        />
      </label>
      {#if session.isOwner}
        <p class="text-sm text-zinc-500 dark:text-zinc-400">
          This doesn't change your password for the admin page.
        </p>
      {/if}
      {#if error}<p class="text-sm text-red-600 dark:text-red-400" role="alert">{error}</p>{/if}
      <button class="btn-primary" disabled={busy}>{busy ? 'Saving…' : 'Change password'}</button>
    </form>
  {/if}
  {#if session.isOwner}
    <p>
      Manage data in the <a class="text-brand-600 underline dark:text-brand-400" href="/_/"
        >admin page</a
      >.
    </p>
  {/if}
</section>
