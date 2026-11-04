# Build your first feature

This guide adds a small feature to a new app, one layer at a time. The feature is
**shared tasks**: people add tasks for each other, the assignee gets a push
notification, everyone gets a reminder each morning, and a worker adds a task when
rain is on the way. It is about 200 lines in all, and each step leaves the app
working.

| Step | Layer | File |
| --- | --- | --- |
| 1 | The data and who may touch it | `backend/migrations/…_tasks.go` |
| 2 | The screen | `web/src/components/Tasks.svelte` |
| 3 | A notification | `backend/app/tasks.go` |
| 4 | A scheduled job | `backend/app/tasks.go` |
| 5 | A worker | `workers/rain.py` |
| 6 | Tests | `backend/app/tasks_test.go`, `workers/tests/test_rain.py` |
| 7 | Ship it | |

<!-- TEMPLATE start -->
## 0. Start the app

On GitHub, open plasto and click **Use this template**, then:

```sh
git clone git@github.com:you/chores.git && cd chores
scripts/rename.sh        # plasto becomes "chores", from the repository's name
scripts/strip-demo.sh    # removes the demo items list
make setup && make dev   # open http://localhost:5173 and create your account with the setup code it prints
```

Commit after each script, so `git diff` shows only your own work from then on.

<!-- TEMPLATE end -->

## 1. The data

Collections are defined in Go migrations, so every copy of the app (your laptop,
the server, the tests) has the same schema. Create one:

```sh
cd backend && go run . migrate create tasks
```

and fill in the new file in `backend/migrations/`:

```go
package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
	"github.com/pocketbase/pocketbase/tools/types"
)

func init() {
	m.Register(func(app core.App) error {
		users, err := app.FindCollectionByNameOrId("users")
		if err != nil {
			return err
		}
		tasks := core.NewBaseCollection("tasks")
		tasks.Fields.Add(
			&core.TextField{Name: "title", Required: true, Max: 200},
			&core.BoolField{Name: "done"},
			&core.RelationField{Name: "assignee", CollectionId: users.Id, MaxSelect: 1},
			&core.AutodateField{Name: "created", OnCreate: true},
			&core.AutodateField{Name: "updated", OnCreate: true, OnUpdate: true},
		)
		signedIn := types.Pointer(`@request.auth.id != ""`)
		tasks.ListRule, tasks.ViewRule, tasks.CreateRule, tasks.UpdateRule = signedIn, signedIn, signedIn, signedIn
		tasks.DeleteRule = types.Pointer(`@request.auth.role = "owner"`)
		return app.Save(tasks)
	}, func(app core.App) error {
		tasks, err := app.FindCollectionByNameOrId("tasks")
		if err != nil {
			return err
		}
		return app.Delete(tasks)
	})
}
```

The rules are the whole access policy; there is no other permission code. Each is a
[filter](https://pocketbase.io/docs/api-rules-and-filters/) the request must match:

| Rule | Who it lets in |
| --- | --- |
| `nil` (the default) | Nobody but superusers |
| `@request.auth.id != ""` | Anyone signed in: people and workers |
| `@request.auth.collectionName = "users"` | People only |
| `@request.auth.collectionName = "workers"` | Workers only |
| `@request.auth.role = "owner"` | The owner |
| `assignee = @request.auth.id` | The person the record belongs to |

Restart `make dev` (or save a Go file) and the migration runs. The collection is
now in the admin UI at http://localhost:8090/_/ and in the API at
`/api/collections/tasks/records`.

## 2. The screen

Add `web/src/components/Tasks.svelte`:

```svelte
<script lang="ts">
  import { onMount } from 'svelte';
  import type { RecordModel } from 'pocketbase';
  import { errorMessage, pb, type User } from '../lib/pb';

  interface Task extends RecordModel {
    title: string;
    done: boolean;
    assignee: string;
  }

  let tasks = $state<Task[]>([]);
  let people = $state<User[]>([]);
  let title = $state('');
  let assignee = $state('');
  let error = $state('');

  const collection = pb.collection<Task>('tasks');
  const nameOf = (id: string) => people.find((p) => p.id === id)?.name || '';

  onMount(() => {
    Promise.all([
      collection.getFullList({ filter: 'done = false', sort: 'created' }),
      pb.collection<User>('users').getFullList({ sort: 'name' }),
    ])
      .then(([t, p]) => ((tasks = t), (people = p)))
      .catch((err) => (error = errorMessage(err)));

    // Changes made by anyone, a person or a worker, arrive here live.
    const unsubscribe = collection.subscribe('*', ({ action, record }) => {
      tasks = tasks.filter((t) => t.id !== record.id);
      if (action !== 'delete' && !record.done) tasks = [...tasks, record];
    });
    return () => unsubscribe.then((stop) => stop());
  });

  async function add(e: SubmitEvent) {
    e.preventDefault();
    error = '';
    try {
      await collection.create({ title, assignee });
      title = '';
    } catch (err) {
      error = errorMessage(err);
    }
  }

  async function finish(task: Task) {
    try {
      await collection.update(task.id, { done: true });
    } catch (err) {
      error = errorMessage(err);
    }
  }
</script>

<section class="card flex flex-col gap-3">
  <h2 class="font-semibold">Tasks</h2>
  <form class="flex flex-col gap-2" onsubmit={add}>
    <input placeholder="What needs doing?" bind:value={title} required maxlength="200" />
    <div class="flex gap-2">
      <select bind:value={assignee} aria-label="Assign to">
        <option value="">Anyone</option>
        {#each people as person (person.id)}
          <option value={person.id}>{person.name || person.email}</option>
        {/each}
      </select>
      <button class="btn-primary shrink-0">Add</button>
    </div>
  </form>
  {#if error}<p class="text-sm text-red-600 dark:text-red-400" role="alert">{error}</p>{/if}
  <ul class="divide-y divide-zinc-200 dark:divide-zinc-800">
    {#each tasks as task (task.id)}
      <li class="flex items-center gap-3 py-2">
        <input type="checkbox" onchange={() => finish(task)} aria-label="Done" />
        <span class="flex-1">{task.title}</span>
        <span class="text-xs text-zinc-500">{nameOf(task.assignee)}</span>
      </li>
    {:else}
      <li class="py-2 text-sm text-zinc-500">Nothing to do.</li>
    {/each}
  </ul>
</section>
```

and show it in `web/src/components/Home.svelte`, after the "Your app starts here"
card:

```svelte
<script lang="ts">
  // …
  import Tasks from './Tasks.svelte';
</script>

<!-- … -->
<Tasks />
```

The page updates as you save. Open it in a second browser window, signed in as
someone else, to see tasks arrive live. Inputs, buttons (`btn-primary`,
`btn-secondary`) and `card` are styled in `web/src/app.css`; the rest is
[Tailwind](https://tailwindcss.com/docs).

## 3. A notification

Server-side behavior goes in hooks. Add `backend/app/tasks.go`:

```go
package app

import (
	"github.com/pocketbase/pocketbase/core"

	"github.com/selfhostit/plasto/push"
)

func registerTasks(app core.App) {
	// Tell the assignee about a new task, unless they made it themselves.
	app.OnRecordCreateRequest("tasks").BindFunc(func(e *core.RecordRequestEvent) error {
		if err := e.Next(); err != nil {
			return err
		}
		assignee := e.Record.GetString("assignee")
		if assignee != "" && (e.Auth == nil || e.Auth.Id != assignee) {
			push.ToUser(e.App, assignee, push.Message{
				Title: "New task for you",
				Body:  e.Record.GetString("title"),
				URL:   "/",
			})
		}
		return nil
	})
}
```

and call it from `Register` in `backend/app/app.go`:

```go
	registerTasks(app)
```

`e.Next()` saves the record; code after it runs only when the save worked. Sending
happens in the background, so a slow push service never slows the request. People
get notifications on the devices where they turned them on.

## 4. A scheduled job

Add a morning reminder to `registerTasks`. `jobs.MustAdd` runs on a cron schedule
in the server's time zone, records every run in the admin UI (the `jobs`
collection), and runs a job once at startup if the server was off when it was due.

```go
	// Every morning, remind each person how many tasks they have open.
	jobs.MustAdd(app, "task-reminders", "0 8 * * *", func(app core.App) error {
		open, err := app.FindAllRecords("tasks",
			dbx.HashExp{"done": false}, dbx.Not(dbx.HashExp{"assignee": ""}))
		if err != nil {
			return err
		}
		count := map[string]int{}
		for _, t := range open {
			count[t.GetString("assignee")]++
		}
		for user, n := range count {
			push.ToUser(app, user, push.Message{Title: fmt.Sprintf("%d open tasks", n), URL: "/"})
		}
		return nil
	})
```

with `"fmt"`, `"github.com/pocketbase/dbx"` and `"github.com/selfhostit/plasto/jobs"`
added to the imports.

## 5. A worker

Work that needs Python, other machines, or a lot of time goes in a worker. A worker
signs in with its own account and uses the same API and rules as the web app.
Our rules say `@request.auth.id != ""`, which already lets workers in.

Add `workers/rain.py`:

```python
"""Adds a task when rain is on the way, from the Open-Meteo forecast (no key needed)."""

import sys

import httpx

from plasto_worker import Client, every

HOME = {"latitude": 37.98, "longitude": 23.73}


def rain_chance() -> int:
    """The highest chance of rain in the next six hours, in percent."""
    res = httpx.get(
        "https://api.open-meteo.com/v1/forecast",
        params={**HOME, "hourly": "precipitation_probability", "forecast_hours": 6},
    )
    res.raise_for_status()
    return max(res.json()["hourly"]["precipitation_probability"])


def check(app: Client) -> None:
    if rain_chance() < 60:
        return
    if not app.list("tasks", filter='title = "Bring in the laundry" && done = false'):
        app.create("tasks", {"title": "Bring in the laundry"})


def main(once: bool = False) -> None:
    app = Client()
    every(30 * 60, lambda: check(app), app, once=once)


if __name__ == "__main__":
    main(once="--once" in sys.argv)
```

Create a worker account in the admin UI (the **workers** collection, a name and a
password) and run it:

```sh
cd workers
PLASTO_URL=http://localhost:8090 PLASTO_WORKER=rain PLASTO_WORKER_PASSWORD=... uv run python rain.py
```

Add Python libraries with `uv add <package>`; they go into the worker image too.

## 6. Tests

Go tests start a fresh app with an empty database for each test, run your
migrations and hooks, and send real requests. Add `backend/app/tasks_test.go`:

```go
package app_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tests"
)

func TestPeopleAddTasksForEachOther(t *testing.T) {
	a := newApp(t)
	_, ownerToken := addUser(t, a, "owner@example.com", "owner")
	member, _ := addUser(t, a, "member@example.com", "member")

	s := tests.ApiScenario{
		Method:          http.MethodPost,
		URL:             "/api/collections/tasks/records",
		Body:            strings.NewReader(`{"title":"Water the plants","assignee":"` + member.Id + `"}`),
		Headers:         map[string]string{"Authorization": ownerToken},
		ExpectedStatus:  http.StatusOK,
		ExpectedContent: []string{`"title":"Water the plants"`},
		TestAppFactory:  func(testing.TB) *tests.TestApp { return a },
	}
	s.Test(t)
}

func TestMembersCantDeleteTasks(t *testing.T) {
	a := newApp(t)
	_, memberToken := addUser(t, a, "member@example.com", "member")
	tasks, _ := a.FindCollectionByNameOrId("tasks")
	task := core.NewRecord(tasks)
	task.Set("title", "Water the plants")
	if err := a.Save(task); err != nil {
		t.Fatal(err)
	}

	s := tests.ApiScenario{
		Method:          http.MethodDelete,
		URL:             "/api/collections/tasks/records/" + task.Id,
		Headers:         map[string]string{"Authorization": memberToken},
		ExpectedStatus:  http.StatusNotFound, // a rule that fails hides the record
		ExpectedContent: []string{`"status":404`},
		TestAppFactory:  func(testing.TB) *tests.TestApp { return a },
	}
	s.Test(t)
}
```

Worker tests run against a real server built from `backend/`, with a fresh worker
account per test. Replace the parts that reach the outside world. Add
`workers/tests/test_rain.py`:

```python
import rain
from plasto_worker import Client


def test_rain_adds_the_task_once(server, worker, monkeypatch):
    monkeypatch.setattr(rain, "rain_chance", lambda: 80)
    with Client(server, *worker) as app:
        rain.check(app)
        rain.check(app)
        assert len(app.list("tasks", filter='title = "Bring in the laundry"')) == 1
```

Run everything with `make check test`.

## 7. Ship it

Open a pull request. CI runs the checks and tests, builds both images and deploys
them with Docker Compose for a smoke test. When you merge, it publishes the images.
Then, on the server:

```sh
cd ~/plasto && docker compose pull && docker compose up -d
```

The new migration runs when the app starts. To run the rain worker there, create
its account in the admin UI and set these in `.env`, then run `docker compose up -d`
again:

```sh
COMPOSE_PROFILES=worker
PLASTO_WORKER=rain
PLASTO_WORKER_PASSWORD=...
PLASTO_WORKER_SCRIPT=rain.py
```

See [deploy.md](deploy.md) for the rest.
