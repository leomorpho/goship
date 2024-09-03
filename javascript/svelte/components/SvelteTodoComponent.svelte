<script>
  let todos = [];
  let newTodo = "";

  function addTodo() {
    if (newTodo.trim()) {
      todos = [...todos, { id: Date.now(), text: newTodo, done: false }];
      newTodo = "";
    }
  }

  function removeTodo(id) {
    todos = todos.filter(todo => todo.id !== id);
  }

  function toggleDone(id) {
    todos = todos.map(todo =>
      todo.id === id ? { ...todo, done: !todo.done } : todo
    );
  }
</script>

<div class="max-w-md mx-auto my-10 bg-white dark:bg-gray-900 text-black dark:text-white shadow-lg rounded-lg overflow-hidden">
  <div class="px-6 py-4">
    <div class="font-bold text-xl mb-2">Todos</div>
    <div class="flex justify-between mb-4">
      <input
        class="shadow appearance-none rounded w-full py-2 px-3 mr-4 text-black dark:text-white bg-gray-100 dark:bg-gray-700"
        bind:value={newTodo}
        on:keydown={event => event.key === 'Enter' && addTodo()}
        placeholder="Add new todo"
      />
      <button
        class="flex-no-shrink p-2 rounded-lg text-white bg-teal-500 hover:bg-teal-400 transition-colors duration-200 ease-in-out"
        on:click={addTodo}
      >
        Add
      </button>
    </div>

    {#each todos as todo}
      <div class="flex items-center justify-between mb-4 bg-gray-100 dark:bg-gray-700 text-black dark:text-white rounded p-2">
        <div class="flex items-center">
          <input type="checkbox" class="mr-2 leading-tight" bind:checked={todo.done} on:click={() => toggleDone(todo.id)} />
          <p class="{todo.done ? 'text-gray-500 line-through' : 'text-gray-900'}">
            {todo.text}
          </p>
        </div>
        <button
          class="flex-no-shrink p-2 rounded-lg text-white bg-red-500 hover:bg-red-400 transition-colors duration-200 ease-in-out"
          on:click={() => removeTodo(todo.id)}
        >
          Remove
        </button>
      </div>
    {/each}
  </div>
</div>

<style>
  .line-through {
    text-decoration: line-through;
  }
</style>
