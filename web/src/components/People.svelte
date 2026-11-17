<script lang="ts">
  import { onMount } from 'svelte';
  import { errorMessage, pb, type User } from '../lib/pb';
  import { generatePassword, inviteText, personError } from '../lib/people';
  import { session } from '../lib/session.svelte';

  // The owner adds, resets and removes people here. Only the owner gets here (App.svelte),
  // and the users collection's API rules hold for everyone else anyway.
  let people = $state<User[]>([]);
  let loading = $state(true);
  let error = $state('');

  let name = $state('');
  let email = $state('');
  let busy = $state(false);
  let formError = $state('');

  // what to send to the person just added or reset; the password is never shown again
  let invite = $state<{ text: string; title: string } | null>(null);
  let copied = $state(false);
  let textarea = $state<HTMLTextAreaElement>();
  const canShare = typeof navigator.share === 'function';

  async function load() {
    try {
      people = await pb.collection('users').getFullList<User>({ sort: 'created' });
      error = '';
    } catch (err) {
      error = errorMessage(err);
    } finally {
      loading = false;
    }
  }

  onMount(load);

  function show(person: { name: string; email: string }, password: string, title: string) {
    invite = {
      title,
      text: inviteText({
        appName: document.title,
        address: location.origin,
        email: person.email,
        password,
        name: person.name,
      }),
    };
    copied = false;
    scrollTo({ top: 0 });
  }

  async function add(e: SubmitEvent) {
    e.preventDefault();
    busy = true;
    formError = '';
    const password = generatePassword();
    const person = { name: name.trim(), email: email.trim().toLowerCase() };
    try {
      await pb.collection('users').create({
        ...person,
        role: 'member',
        verified: true,
        password,
        passwordConfirm: password,
      });
      name = email = '';
      show(person, password, `${person.name || person.email} can sign in now`);
      await load();
    } catch (err) {
      formError = personError(err);
    } finally {
      busy = false;
    }
  }

  async function reset(person: User) {
    if (
      !confirm(
        `Make a new password for ${person.name || person.email}? Their old one stops working.`,
      )
    )
      return;
    error = '';
    const password = generatePassword();
    try {
      await pb.collection('users').update(person.id, { password, passwordConfirm: password });
      show(person, password, `New password for ${person.name || person.email}`);
    } catch (err) {
      error = personError(err);
    }
  }

  async function remove(person: User) {
    if (!confirm(`Remove ${person.name || person.email}? They can't sign in anymore.`)) return;
    error = '';
    try {
      await pb.collection('users').delete(person.id);
      await load();
    } catch (err) {
      error = errorMessage(err, `Could not remove ${person.name || person.email}.`);
    }
  }

  async function copy() {
    try {
      await navigator.clipboard.writeText(invite!.text);
      copied = true;
    } catch {
      // no clipboard over plain http: select the text so a long press copies it
      textarea?.select();
    }
  }

  async function share() {
    try {
      await navigator.share({ text: invite!.text });
    } catch {
      // closed the share sheet
    }
  }
</script>

<main
  class="mx-auto flex min-h-dvh max-w-md flex-col gap-4 px-4 pt-[max(1.5rem,env(safe-area-inset-top))] pb-8"
>
  <header class="flex items-center justify-between">
    <h1 class="text-2xl font-bold tracking-tight">People</h1>
    <a class="btn-secondary px-3 py-2 text-sm" href="#/">Back</a>
  </header>

  {#if invite}
    <section class="card flex flex-col gap-3 border-brand-500 dark:border-brand-400">
      <h2 class="font-semibold">{invite.title}</h2>
      <p class="text-sm text-zinc-600 dark:text-zinc-400">
        Send them this. The password is only shown now, so send it before you close this.
      </p>
      <textarea
        bind:this={textarea}
        readonly
        rows="8"
        class="text-sm"
        value={invite.text}
        onfocus={(e) => e.currentTarget.select()}></textarea>
      <div class="flex gap-2">
        <button class="btn-primary flex-1" onclick={copy}>{copied ? 'Copied' : 'Copy'}</button>
        {#if canShare}<button class="btn-secondary flex-1" onclick={share}>Share</button>{/if}
        <button class="btn-secondary flex-1" onclick={() => (invite = null)}>Done</button>
      </div>
    </section>
  {/if}

  <section class="card flex flex-col gap-3">
    <h2 class="font-semibold">Everyone</h2>
    {#if loading}
      <p class="text-sm text-zinc-500">Loading…</p>
    {:else}
      <ul class="divide-y divide-zinc-200 dark:divide-zinc-800">
        {#each people as person (person.id)}
          {@const you = person.id === session.user?.id}
          <li class="flex flex-col gap-2 py-3 first:pt-0 last:pb-0">
            <div class="min-w-0">
              <p class="truncate font-medium">
                {person.name || person.email}
                {#if you}<span class="text-sm font-normal text-zinc-500">(you)</span>{/if}
              </p>
              <p class="truncate text-sm text-zinc-500 dark:text-zinc-400">
                {[person.name && person.email, person.role === 'owner' ? 'Owner' : 'Member']
                  .filter(Boolean)
                  .join(' · ')}
              </p>
            </div>
            {#if !you && person.role !== 'owner'}
              <div class="flex gap-2">
                <button class="btn-secondary px-3 py-2 text-sm" onclick={() => reset(person)}
                  >New password</button
                >
                <button
                  class="btn-secondary px-3 py-2 text-sm text-red-600 dark:text-red-400"
                  onclick={() => remove(person)}>Remove</button
                >
              </div>
            {/if}
          </li>
        {/each}
      </ul>
    {/if}
    {#if error}<p class="text-sm text-red-600 dark:text-red-400" role="alert">{error}</p>{/if}
  </section>

  <section class="card">
    <h2 class="font-semibold">Add someone</h2>
    <p class="mt-1 text-sm text-zinc-600 dark:text-zinc-400">
      They get a password from you and can change it after they sign in.
    </p>
    <form class="mt-4 flex flex-col gap-4" onsubmit={add}>
      <label>Name <input autocomplete="off" bind:value={name} maxlength="100" /></label>
      <label>
        Email
        <input type="email" autocomplete="off" bind:value={email} required />
      </label>
      {#if formError}
        <p class="text-sm text-red-600 dark:text-red-400" role="alert">{formError}</p>
      {/if}
      <button class="btn-primary" disabled={busy}>{busy ? 'Adding…' : 'Add'}</button>
    </form>
  </section>
</main>
