<script lang="ts">
  import Placeholder from "@tiptap/extension-placeholder";
  import StarterKit from "@tiptap/starter-kit";
  import cx from "clsx";
  import { afterUpdate, onMount } from "svelte";

  import {
    BubbleMenu,
    Editor,
    EditorContent,
    FloatingMenu,
    createEditor,
  } from "svelte-tiptap";

  import type { Readable } from "svelte/store";
  import BoldIcon from "./TextEditor/BoldIcon.svelte";
  import ItalicIcon from "./TextEditor/ItalicIcon.svelte";

  export let answerContentProp = "";
  export let answerUpdatedEvent = "AnswerUpdatedEvent";
  export let hiddenAnswerFieldName = "text";
  export let isPaymentsEnabled = true;
  export let numAnswersCanPublish = 1;
  export let cantPublishBecauseWaitingQuestionsNumTooHigh = false;

  $: cannotPublish =
    (numAnswersCanPublish == 0 && isPaymentsEnabled) ||
    isSaving ||
    cantPublishBecauseWaitingQuestionsNumTooHigh;

  // NOTE: the component emitting the event, since it is here meant
  // to be consumed by HTMX, must exist at the time HTMX is initialized.
  // For that reason, pass the ID of an element generated on the BE that
  // is properly placed in relation to the HTMX consumer.
  export let componentIdEmittingEvent = "whoever-wants-to-emit-the-event";

  let debounceTimer: any;
  const debouncePeriod = 500; // Debounce period in milliseconds
  let canDispatch = true; // Flag to control event dispatch within debounce period

  let editor: Readable<Editor>;
  let editorContent = answerContentProp || ""; // This store will hold the editor's content
  let isSaving = false;
  let lastDispatchedContent = ""; // Initialize with the initial content or empty string

  onMount(() => {
    editor = createEditor({
      extensions: [
        StarterKit,
        Placeholder.configure({
          placeholder: "Write your answer here 🖋",
        }),
      ],
      content: answerContentProp || `<p></p>`,
      editorProps: {
        attributes: {
          class:
            "border-2 border-slate-500 dark:border-slate-800 rounded-md p-3 outline-none",
        },
        handleDOMEvents: {
          focus: (view, event) => {
            // Ensure the editor is focused when interacting
            view.focus();
            return false;
          },
        },
      },
      onUpdate: ({ editor }) => {
        editorContent = editor.getHTML(); // Update the store with the latest content
      },
    });
  });

  const toggleBold = () => {
    $editor.chain().focus().toggleBold().run();
  };

  const toggleItalic = () => {
    $editor.chain().focus().toggleItalic().run();
  };

  $: isActive = (name: string, attrs = {}) => $editor.isActive(name, attrs);

  function dispatchAnswerUpdatedEvent() {
    const dropdownElement = document.getElementById(componentIdEmittingEvent);
    if (dropdownElement) {
      isSaving = true;
      setTimeout(() => {
        isSaving = false;
      }, 500); // Display "Saving..." for 0.5 seconds
      const customEvent = new CustomEvent(answerUpdatedEvent, {
        detail: {},
      });
      console.log("DISPATCH EVENT");
      dropdownElement.dispatchEvent(customEvent);
    }
  }

  // Dispatch the event after updates, specifically after hidden inputs have been added
  afterUpdate(() => {
    if (
      editorContent !== lastDispatchedContent &&
      canDispatch &&
      editorContent.length > 0
    ) {
      dispatchAnswerUpdatedEvent();
      lastDispatchedContent = editorContent;
      canDispatch = false; // Prevent further dispatches
      // Reset canDispatch flag after the debounce period
      clearTimeout(debounceTimer);
      debounceTimer = setTimeout(() => {
        canDispatch = true;
      }, debouncePeriod);
    }
  });
</script>

{#if editor}
  <BubbleMenu editor={$editor}>
    <div data-test-id="bubble-menu" class="flex items-center justify-center">
      <button
        class={cx(
          "border border-black rounded px-2 mr-1 hover:bg-black hover:text-white",
          {
            "bg-black text-white": isActive("bold"),
          }
        )}
        type="button"
        on:click={toggleBold}
      >
        <BoldIcon />
      </button>
      <button
        class={cx(
          "border border-black rounded px-2 hover:bg-black hover:text-white",
          {
            "bg-black text-white": isActive("italic"),
          }
        )}
        type="button"
        on:click={toggleItalic}
      >
        <ItalicIcon />
      </button>
    </div>
  </BubbleMenu>
  <FloatingMenu editor={$editor}>
    <div data-test-id="floating-menu" class="flex items-center justify-center">
      <button
        class={cx(
          "border border-black rounded px-2 mr-1 hover:bg-black hover:text-white",
          {
            "bg-black text-white": isActive("bold"),
          }
        )}
        type="button"
        on:click={toggleBold}
      >
        <BoldIcon />
      </button>
      <button
        class={cx(
          "border border-black rounded px-2 hover:bg-black hover:text-white",
          {
            "bg-black text-white": isActive("italic"),
          }
        )}
        type="button"
        on:click={toggleItalic}
      >
        <ItalicIcon />
      </button>
    </div>
  </FloatingMenu>
{/if}
<EditorContent editor={$editor} />
<div class="button-container flex justify-center mt-4 md:mt-8 lg:mt-12 mb-10">
  <button
    class="bg-green-500 hover:bg-green-700 opacity-70 text-white font-bold w-32 md:w-40 py-4 px-3 mx-2 rounded-lg"
    class:opacity-50={cannotPublish}
    class:cursor-not-allowed={cannotPublish}
    class:bg-green-700={cannotPublish}
    class:bg-green-600={cannotPublish}
    disabled={cannotPublish}
  >
    Publish
  </button>
</div>
<!-- Hidden input field to store the editor's content -->
<input type="hidden" name={hiddenAnswerFieldName} bind:value={editorContent} />
