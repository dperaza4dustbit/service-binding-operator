package binding

import (
	"testing"

	"github.com/redhat-developer/service-binding-operator/test/mocks"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestStringDefinition(t *testing.T) {
	type args struct {
		description   string
		outputName    string
		path          string
		value         string
		expectedValue interface{}
	}

	testCases := []args{
		{
			description: "outputName informed",
			outputName:  "username",
			path:        "{.status.dbCredentials.username}",
			expectedValue: map[string]interface{}{
				"username": "AzureDiamond",
			},
		},
		{
			description: "outputName informed - alias",
			outputName:  "anotherName",
			path:        "{.status.dbCredentials.username}",
			expectedValue: map[string]interface{}{
				"anotherName": "AzureDiamond",
			},
		},
		{
			description: "raw value",
			outputName:  "foo",
			value:       "bar",
			expectedValue: map[string]interface{}{
				"foo": "bar",
			},
		},
		{
			description: "value with prefix",
			outputName:  "foo",
			path:        "foo-{.status.dbCredentials.username}",
			expectedValue: map[string]interface{}{
				"foo": "foo-AzureDiamond",
			},
		},
		{
			description: "concatenate two json paths",
			outputName:  "foo",
			path:        "{.status.dbCredentials.username}-{.status.dbCredentials.password}",
			expectedValue: map[string]interface{}{
				"foo": "AzureDiamond-foo",
			},
		},
	}

	u := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"status": map[string]interface{}{
				"dbCredentials": map[string]interface{}{
					"username": "AzureDiamond",
					"password": "foo",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			d := &stringDefinition{
				outputName: tc.outputName,
				definition: definition{
					path: tc.path,
				},
				value: tc.value,
			}
			val, err := d.Apply(u)
			require.NoError(t, err)
			require.Equal(t, tc.expectedValue, val.Get())
		})
	}
}

func TestStringOfMap(t *testing.T) {
	type args struct {
		description   string
		outputName    string
		path          string
		expectedValue interface{}
		object        *unstructured.Unstructured
	}

	u := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"status": map[string]interface{}{
				"dbCredentials": map[string]interface{}{
					"username": "AzureDiamond",
					"password": "hunter2",
				},
			},
		},
	}

	expectedValue := map[string]interface{}{
		"dbCredentials": map[string]interface{}{
			"username": "AzureDiamond",
			"password": "hunter2",
		},
	}

	testCases := []args{
		{
			description:   "outputName informed",
			expectedValue: expectedValue,
			object:        u,
			outputName:    "dbCredentials",
			path:          "{.status.dbCredentials}",
		},
		{
			description: "outputName empty",
			expectedValue: map[string]interface{}{
				"username": "AzureDiamond",
				"password": "hunter2",
			},
			object: u,
			path:   "{.status.dbCredentials}",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			d := &stringOfMapDefinition{
				outputName: tc.outputName,
				definition: definition{
					path: tc.path,
				},
			}
			val, err := d.Apply(tc.object)
			require.NoError(t, err)
			require.Equal(t, tc.expectedValue, val.Get())
		})
	}
}

func TestSliceOfStringsFromPath(t *testing.T) {
	d := &sliceOfStringsFromPathDefinition{
		sourceValue: "url",
		definition: definition{
			path: "{.status.bootstrap}",
		},
		outputName: "bootstrap",
	}
	val, err := d.Apply(&unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"namespace": "test-namespace",
			},
			"status": map[string]interface{}{
				"bootstrap": []interface{}{
					map[string]interface{}{
						"type": "http",
						"url":  "www.example.com",
					},
					map[string]interface{}{
						"type": "https",
						"url":  "secure.example.com",
					},
				},
			},
		},
	})
	require.NoError(t, err)
	v := map[string]interface{}{
		"bootstrap": []interface{}{"www.example.com", "secure.example.com"},
	}
	require.Equal(t, v, val.Get())
}

func TestSliceOfMapsFromPath(t *testing.T) {
	d := &sliceOfMapsFromPathDefinition{
		sourceKey:   "type",
		sourceValue: "url",
		outputName:  "bootstrap",
		definition: definition{
			path: "{.status.bootstrap}",
		},
	}
	val, err := d.Apply(&unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"namespace": "test-namespace",
			},
			"status": map[string]interface{}{
				"bootstrap": []interface{}{
					map[string]interface{}{
						"type": "http",
						"url":  "www.example.com",
					},
					map[string]interface{}{
						"type": "https",
						"url":  "secure.example.com",
					},
				},
			},
		},
	})
	require.NoError(t, err)
	v := map[string]interface{}{
		"bootstrap": map[string]interface{}{
			"http":  "www.example.com",
			"https": "secure.example.com",
		},
	}
	require.Equal(t, v, val.Get())
}

func TestMapFromSecretDataField(t *testing.T) {
	f := mocks.NewFake(t, "test-namespace")
	f.AddMockedUnstructuredSecret("dbCredentials-secret")
	d := &mapFromDataFieldDefinition{
		secretConfigMapReader: &secretConfigMapReader{
			secretReader:    secretsReader(f.FakeDynClient()),
			configMapReader: configMapsReader(f.FakeDynClient()),
		},
		objectType: secretObjectType,
		definition: definition{
			path: "{.status.dbCredentials}",
		},
	}
	val, err := d.Apply(&unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"namespace": "test-namespace",
			},
			"status": map[string]interface{}{
				"dbCredentials": "dbCredentials-secret",
			},
		},
	})
	require.NoError(t, err)
	v := map[string]string{
		"username": "user",
		"password": "password",
	}
	require.Equal(t, v, val.Get())
}

func TestMapFromConstructedSecretName(t *testing.T) {
	f := mocks.NewFake(t, "test-namespace")
	f.AddMockedUnstructuredSecret("foo-dbCredentials-secret")
	d := &mapFromDataFieldDefinition{
		secretConfigMapReader: &secretConfigMapReader{
			secretReader:    secretsReader(f.FakeDynClient()),
			configMapReader: configMapsReader(f.FakeDynClient()),
		},
		objectType: secretObjectType,
		definition: definition{
			path: "foo-{.status.dbCredentials}",
		},
	}
	val, err := d.Apply(&unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"namespace": "test-namespace",
			},
			"status": map[string]interface{}{
				"dbCredentials": "dbCredentials-secret",
			},
		},
	})
	require.NoError(t, err)
	v := map[string]string{
		"username": "user",
		"password": "password",
	}
	require.Equal(t, v, val.Get())
}

func TestMapFromConfigMapDataField(t *testing.T) {
	f := mocks.NewFake(t, "test-namespace")
	f.AddMockedUnstructuredConfigMap("dbCredentials-configMap")
	d := &mapFromDataFieldDefinition{
		secretConfigMapReader: &secretConfigMapReader{
			secretReader:    secretsReader(f.FakeDynClient()),
			configMapReader: configMapsReader(f.FakeDynClient()),
		},
		objectType: configMapObjectType,
		definition: definition{
			path: "{.status.dbCredentials}",
		},
	}
	val, err := d.Apply(&unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"namespace": "test-namespace",
			},
			"status": map[string]interface{}{
				"dbCredentials": "dbCredentials-configMap",
			},
		},
	})
	require.NoError(t, err)
	v := map[string]string{
		"username": "user",
		"password": "password",
	}
	require.Equal(t, v, val.Get())
}

func TestMapFromConfigMapDataFieldWithOutputNameAndSourceValue(t *testing.T) {
	f := mocks.NewFake(t, "test-namespace")
	f.AddMockedUnstructuredConfigMap("dbCredentials-configMap")
	d := &mapFromDataFieldDefinition{
		secretConfigMapReader: &secretConfigMapReader{
			secretReader:    secretsReader(f.FakeDynClient()),
			configMapReader: configMapsReader(f.FakeDynClient()),
		},
		objectType:  configMapObjectType,
		sourceValue: "username",
		outputName:  "user",
		definition: definition{
			path: "{.status.dbCredentials}",
		},
	}
	val, err := d.Apply(&unstructured.Unstructured{
		Object: map[string]interface{}{
			"metadata": map[string]interface{}{
				"namespace": "test-namespace",
			},
			"status": map[string]interface{}{
				"dbCredentials": "dbCredentials-configMap",
			},
		},
	})
	require.NoError(t, err)
	v := map[string]string{
		"user": "user",
	}
	require.Equal(t, v, val.Get())
}
