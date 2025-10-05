<template>
  <div class="page-container">
    <h1>Update Employee</h1>

    <div v-if="loading" class="info">Loading...</div>
    <div v-else>
      <div class="form-group">
        <input v-model="empName" type="text" placeholder="Enter an employee Name" />
      </div>
      <div class="form-group">
        <input v-model="department" type="text" placeholder="Enter department" />
      </div>
      <div class="form-group">
        <input v-model="language" type="text" placeholder="Enter language" />
      </div>

      <div class="button-group">
        <button @click="updateEmployee" :disabled="!canUpdate">Update</button>
        <button @click="cancel">Cancel</button>
      </div>

      <div v-if="error" class="error">{{ error }}</div>
      <div v-if="success" class="success">{{ success }}</div>

      <!-- Back to Home button -->
      <button class="back-btn" @click="goHome"> Back to Home</button>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue';
import axios from 'axios';
import { useRouter, useRoute } from 'vue-router';

const router = useRouter();
const route = useRoute();

const empName = ref('');
const department = ref('');
const language = ref('');
const error = ref(null);
const success = ref(null);
const loading = ref(true);
const original = ref({ emp_name: '', department: '', language: '' });

const API_URL = 'http://localhost:8080/api/employees';

// get id from route
const idParam = route.params.id;
if (!idParam) {
  // if no id, redirect back
  router.push('/');
}

const idNum = parseInt(String(idParam), 10);
if (Number.isNaN(idNum)) {
  // invalid id; redirect back
  router.push('/');
}

// helper to go home/details
function goHome() {
  router.push('/');
}

// whether any field changed (enable update)
const canUpdate = computed(() => {
  // require name at least
  return (
    !!empName.value &&
    (empName.value !== original.value.emp_name ||
      department.value !== original.value.department ||
      language.value !== original.value.language)
  );
});

// Fetch employee details and prefill fields
async function fetchEmployee() {
  loading.value = true;
  error.value = null;
  success.value = null;
  try {
    // We don't have a single-get-by-id endpoint, so get all and find
    const res = await axios.get(API_URL);
    const list = Array.isArray(res.data) ? res.data : [];
    // backend returns emp_id as number or string; normalize
    const emp = list.find((e) => {
      // try numeric comparison
      const v = e.emp_id;
      if (typeof v === 'number') return v === idNum;
      if (typeof v === 'string') return parseInt(v, 10) === idNum;
      // other types
      return false;
    });

    if (!emp) {
      error.value = `Employee with id ${idNum} not found`;
      // still allow user to navigate back
      empName.value = '';
      department.value = '';
      language.value = '';
      original.value = { emp_name: '', department: '', language: '' };
      loading.value = false;
      return;
    }

    // prefill
    empName.value = emp.emp_name || '';
    // note: aggregated fields may be present under lowercase keys (department, language)
    department.value = emp.department ?? emp.Department ?? '';
    language.value = emp.language ?? emp.Language ?? '';

    // save original for change detection
    original.value = {
      emp_name: empName.value,
      department: department.value,
      language: language.value,
    };
  } catch (err) {
    console.error(err);
    error.value = err.response?.data?.error || err.message || 'Failed to fetch employee';
  } finally {
    loading.value = false;
  }
}

// Call update API
async function updateEmployee() {
  error.value = null;
  success.value = null;

  // Basic validation
  if (!empName.value) {
    error.value = 'Employee name is required';
    return;
  }

  try {
    const payload = {
      emp_name: empName.value,
      department: department.value,
      language: language.value,
    };

    // PUT /api/employees/:id
    await axios.put(`${API_URL}/${idNum}`, payload);

    success.value = `Employee ${empName.value} updated successfully`;
    // update original so canUpdate becomes false
    original.value = {
      emp_name: empName.value,
      department: department.value,
      language: language.value,
    };

    // Optionally redirect back to list after a short delay
    setTimeout(() => {
      router.push('/employee-details');
    }, 800);
  } catch (err) {
    console.error(err);
    error.value = err.response?.data?.error || err.message || 'Failed to update employee';
  }
}

// Cancel - return to list
function cancel() {
  router.push('/employee-details');
}


onMounted(() => {
  fetchEmployee();
});
</script>

<style scoped>
.page-container {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: flex-start;
  min-height: 100vh;
  padding-top: 20px;
}

h1 {
  margin-bottom: 20px;
}

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

/* Update button */
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
button:last-child:hover {
  background-color: #bdbdbd;
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

.error {
  color: red;
  margin-top: 12px;
}
.success {
  color: green;
  margin-top: 12px;
}
.info {
  color: #666;
  margin-bottom: 10px;
}
</style>
