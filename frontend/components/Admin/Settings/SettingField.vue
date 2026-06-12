<script setup lang="ts">
  import { computed } from "vue";
  import { Input } from "@/components/ui/input";
  import { Label } from "@/components/ui/label";
  import { Switch } from "@/components/ui/switch";
  import { Textarea } from "@/components/ui/textarea";
  import FormGoogleFontSelect from "~/components/Form/GoogleFontSelect.vue";

  export type FieldType = "boolean" | "number" | "text" | "secret" | "list" | "durationSeconds" | "googleFont";

  export type FieldDef = {
    /** JSON key within the settings section payload. */
    key: string;
    label: string;
    type: FieldType;
    help?: string;
    placeholder?: string;
    /** Empty text inputs persist as null instead of "" (pointer fields). */
    nullable?: boolean;
  };

  const props = defineProps<{
    def: FieldDef;
    modelValue: unknown;
    viewOnly: boolean;
  }>();

  const emit = defineEmits<{ "update:modelValue": [value: unknown] }>();

  const fieldId = computed(() => `setting-${props.def.key}`);

  const boolValue = computed({
    get: () => props.modelValue === true,
    set: (v: boolean) => emit("update:modelValue", v),
  });

  // Text-ish inputs share one string model; the setter converts back to the
  // wire type the section payload expects.
  const textValue = computed({
    get: () => {
      const mv = props.modelValue;
      switch (props.def.type) {
        case "list":
          return Array.isArray(mv) ? mv.join("\n") : "";
        case "durationSeconds":
          return mv == null ? "" : String((mv as number) / 1e9);
        case "number":
          return mv == null ? "" : String(mv);
        default:
          return mv == null ? "" : String(mv);
      }
    },
    set: (raw: string) => {
      switch (props.def.type) {
        case "list":
          emit(
            "update:modelValue",
            raw
              .split("\n")
              .map(s => s.trim())
              .filter(Boolean)
          );
          break;
        case "durationSeconds":
          emit("update:modelValue", raw === "" ? null : Number(raw) * 1e9);
          break;
        case "number":
          emit("update:modelValue", raw === "" ? 0 : Number(raw));
          break;
        default:
          emit("update:modelValue", raw === "" && props.def.nullable ? null : raw);
      }
    },
  });

  const fontValue = computed({
    get: () => (props.modelValue == null ? "default" : String(props.modelValue)),
    set: (v: string) => emit("update:modelValue", v),
  });

  const readonlyDisplay = computed(() => {
    const mv = props.modelValue;
    if (props.def.type === "boolean") return mv === true ? "✓" : "✗";
    if (props.def.type === "secret") return mv ? "••••••••" : "—";
    if (props.def.type === "list") return Array.isArray(mv) && mv.length ? mv.join(", ") : "—";
    if (props.def.type === "durationSeconds") return mv == null ? "—" : `${(mv as number) / 1e9}s`;
    return mv == null || mv === "" ? "—" : String(mv);
  });
</script>

<template>
  <div v-if="viewOnly" class="flex items-baseline justify-between gap-4 text-sm">
    <span class="text-muted-foreground">{{ def.label }}</span>
    <span class="text-right font-medium">{{ readonlyDisplay }}</span>
  </div>

  <div v-else-if="def.type === 'boolean'" class="flex items-center gap-3">
    <Switch :id="fieldId" v-model="boolValue" />
    <Label :for="fieldId">{{ def.label }}</Label>
  </div>

  <div v-else-if="def.type === 'googleFont'" class="flex flex-col gap-1.5">
    <FormGoogleFontSelect v-model="fontValue" :label="def.label" />
    <p v-if="def.help" class="px-1 text-xs text-muted-foreground">{{ def.help }}</p>
  </div>

  <div v-else class="flex flex-col gap-1.5">
    <Label :for="fieldId" class="px-1">{{ def.label }}</Label>
    <Textarea v-if="def.type === 'list'" :id="fieldId" v-model="textValue" :placeholder="def.placeholder" rows="3" />
    <Input
      v-else
      :id="fieldId"
      v-model="textValue"
      :type="
        def.type === 'secret' ? 'password' : def.type === 'number' || def.type === 'durationSeconds' ? 'number' : 'text'
      "
      :placeholder="def.placeholder"
      autocomplete="off"
    />
    <p v-if="def.help" class="px-1 text-xs text-muted-foreground">{{ def.help }}</p>
  </div>
</template>
