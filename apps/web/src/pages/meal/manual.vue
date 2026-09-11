<script setup lang="ts">
import { onMounted } from 'vue'
import ConfirmPage from './confirm.vue'
import { mealDraftStore } from '../../stores/meal-draft'
onMounted(()=>{mealDraftStore.state.mode='manual';mealDraftStore.state.analysis=null;if(!mealDraftStore.state.items.length)mealDraftStore.state.items=[{draftItemId:null,name:'',grams:100,nutrition:{energyKcal:0,proteinGrams:0,carbGrams:0,fatGrams:0},basisPer100Grams:null,origin:'manual',confidence:null,assumption:null}]})
</script>
<template><ConfirmPage/></template>
