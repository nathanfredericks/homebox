<template>
  <div class="flex flex-col gap-1">
    <Label v-if="label" :for="id" class="px-1">{{ label }}</Label>

    <Popover v-model:open="open">
      <PopoverTrigger as-child>
        <Button :id="id" variant="outline" role="combobox" :aria-expanded="open" class="w-full justify-between">
          <span class="min-w-0 flex-auto truncate text-left">
            {{ isDefault ? $t("components.form.google_font_select.system_default") : value }}
          </span>
          <ChevronsUpDown class="ml-2 size-4 shrink-0 opacity-50" />
        </Button>
      </PopoverTrigger>
      <PopoverContent class="w-[--reka-popper-anchor-width] p-0">
        <Command :ignore-filter="true">
          <CommandInput v-model="search" :placeholder="$t('components.form.google_font_select.search')" />
          <CommandEmpty>{{ $t("components.form.google_font_select.no_results") }}</CommandEmpty>
          <CommandList>
            <CommandGroup>
              <CommandItem value="default" @select="select(DEFAULT_FONT)">
                <Check :class="cn('mr-2 h-4 w-4', isDefault ? 'opacity-100' : 'opacity-0')" />
                {{ $t("components.form.google_font_select.system_default") }}
              </CommandItem>
              <CommandItem
                v-for="font in filteredFonts"
                :key="font.family"
                :value="font.family"
                @select="select(font.family)"
              >
                <Check :class="cn('mr-2 h-4 w-4', value === font.family ? 'opacity-100' : 'opacity-0')" />
                <div class="flex w-full items-center justify-between gap-2">
                  <span class="truncate">{{ font.family }}</span>
                  <span class="shrink-0 text-xs text-muted-foreground">{{ font.category }}</span>
                </div>
              </CommandItem>
            </CommandGroup>
          </CommandList>
        </Command>
      </PopoverContent>
    </Popover>
  </div>
</template>

<script setup lang="ts">
  import { Check, ChevronsUpDown } from "lucide-vue-next";
  import fuzzysort from "fuzzysort";
  import { Button } from "~/components/ui/button";
  import { Command, CommandEmpty, CommandGroup, CommandInput, CommandItem, CommandList } from "~/components/ui/command";
  import { Label } from "~/components/ui/label";
  import { Popover, PopoverContent, PopoverTrigger } from "~/components/ui/popover";
  import { cn } from "~/lib/utils";
  import { googleFonts, type GoogleFontCategory } from "~~/lib/data/google-fonts";
  import { DEFAULT_FONT } from "~~/composables/use-google-font";

  type Props = {
    /** "default" for the system stack, otherwise a Google Font family name. */
    modelValue?: string;
    label?: string;
    /** Restrict the list, e.g. ["Monospace"] for a mono font slot. */
    categories?: GoogleFontCategory[];
  };

  const props = withDefaults(defineProps<Props>(), {
    modelValue: DEFAULT_FONT,
    label: "",
    categories: undefined,
  });
  const emit = defineEmits(["update:modelValue"]);

  const id = useId();
  const open = ref(false);
  const search = ref("");
  const value = useVModel(props, "modelValue", emit);
  const isDefault = computed(() => !value.value || value.value === DEFAULT_FONT);

  const fonts = computed(() =>
    props.categories ? googleFonts.filter(f => props.categories!.includes(f.category)) : googleFonts
  );

  const filteredFonts = computed(() => {
    return fuzzysort.go(search.value, fonts.value, { key: "family", all: true, limit: 50 }).map(result => result.obj);
  });

  function select(family: string) {
    value.value = family;
    open.value = false;
    search.value = "";
  }
</script>
