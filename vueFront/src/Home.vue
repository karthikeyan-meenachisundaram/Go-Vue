<template>
  <div class="page-container">
    <h1>Welcome to Wavicle Data Solutions</h1>
    <div class="button-group">
      <button @click="goToCreateEmployee">Create New Employee</button>
      <button @click="goToEmployeeDetails">Show Employee Details</button> 
    </div>
  </div>
</template>

<script setup>
import { useRouter } from 'vue-router';

const router = useRouter();

function goToCreateEmployee() {
  router.push('/create-employee');
}

function goToEmployeeDetails() {
  router.push('/employee-details');
}
</script>

<style scoped>
.page-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  min-height: 100vh;
  padding: 40px 16px;   /* already there, but we’ll use this for spacing */
}

.button-group {
  display: flex;         /* horizontal layout */
  gap: 20px;             /* space between buttons */
  margin-top: 20px;      /* space below heading */
}

button {
  padding: 10px 20px;
  font-size: 16px;
  cursor: pointer;
}
</style>
