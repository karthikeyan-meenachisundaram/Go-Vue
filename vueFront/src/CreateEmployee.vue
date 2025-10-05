<template>
  <div class="page-container">
    <h1>Create New Employee</h1>
    <div class="form-group">
        <input v-model="empName" type="text" placeholder="Enter an employee Name"/>
    </div>
    <div class="form-group">
      <input v-model="department" type="text" placeholder="Enter department" />
    </div>
    <div class="form-group">
      <input v-model="language" type="text" placeholder="Enter language" />
    </div>

    <div class="button-group">
        <button @click="createEmployee"> Create Employee</button>
        <button @click="cancel" :disabled="!hasInput">Clear</button>
    </div>

    <div v-if="error" class="error">{{ error }}</div>
    <div v-if="success" class="success">{{ success }}</div>
    <!-- BACK TO HOME BUTTON -->
    <button class="back-btn" @click="goHome"> Back to Home</button>
  </div>
</template>

<script setup>

import { computed, ref } from 'vue';
import axios from 'axios';
import { useRouter } from 'vue-router';

const router = useRouter();

const empName = ref('');
const department = ref('');
const language = ref('');
const error = ref(null);
const success = ref(null);


function goHome(){
    router.push('/');
}


// Computed property: true if any input has value
const hasInput = computed(() => {
    return empName.value || department.value || language.value;
})

// API base URL
const API_URL = 'http://localhost:8080/api/employees';

// Create Employee function
async function createEmployee() {
    if(!empName.value || !department.value || !language.value){
        error.value = 'All fields are required';
        success.value = null;
        return;
    }

    error.value = null;
    success.value = null;


    try{
        // Step 1: Get last emp_id from backend
        const resLast = await axios.get(`${API_URL}/last-id`);
        const lastEmpId = resLast.data.last_emp_id || 0;
        const newEmpId = lastEmpId + 1;

        // Step 2: Create new employee
        const newEmployee = {
            emp_id: newEmpId,
            emp_name: empName.value,
            department: department.value,
            language: language.value
        };

        await axios.post(`${API_URL}/create`, newEmployee);

        success.value = `Employee ${empName.value} created successfully with ID ${newEmpId}`;

         // Clear inputs
        empName.value = '';
        department.value = '';
        language.value = '';
    } catch (err) {
        console.error(err);
        error.value = err.response?.data?.error || err.message || 'Failed to create employee';
    }
    
}

// Cancel button navigates back to home
    function cancel() {
       // Clear inputs if any
        empName.value = '';
        department.value = '';
        language.value = '';

    }

</script>

<style scoped>
.page-container {
  display: flex;
  flex-direction: column;
  align-items: center;       
  justify-content: flex-start; 
  min-height: 100vh;         
  padding-top: 0px;         
}

h1 { margin-bottom: 20px; }

.form-group {
  margin-bottom: 16px;
}

input {
  padding: 8px 12px;
  width: 300px;
  font-size: 16px;
  border-radius: 4px;
  border: 1px solid #ccc;
}

.button-group {
  display: flex;
  gap: 12px;
  margin-top: 16px;
}

/* Common button style */
button {
  padding: 8px 16px;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  color: white;
  font-size: 14px;
}

/* Create Employee button */
button:first-child {
  background-color: #1976d2;
}
button:first-child:hover {
  background-color: #1565c0;
}
/* Cancel button */
button:last-child {
  background-color: #ccc;
  color: #333;
}
button:last-child:enabled {
  background-color: #1976d2; /* blue when enabled */
  color: white;
}
button:last-child:enabled:hover {
  background-color: #1565c0;
}
button:last-child:disabled {
  cursor: not-allowed;
}

/* Back to Home button */
.back-btn {
  margin-top: 20px;
  padding: 8px 16px;
  background-color: #1976d2;
  color: white;
  border: none;
  border-radius: 4px;
  cursor: pointer;
}

.back-btn:hover {
  background-color: #1565c0;
}

.error { color: red; margin-top: 12px; }
.success { color: green; margin-top: 12px; }


</style>
