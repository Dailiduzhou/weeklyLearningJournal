use proptest::prelude::*;
use property_test::reverse;

proptest! {
    #[test]
    fn test_reverse_twice_is_identity(v in prop::collection::vec(any::<i32>(), 0..100)) {
        // Property: reversing twice gives back the original
        assert_eq!(reverse(&reverse(&v)), v);
    }

    #[test]
    fn test_reverse_preserves_length(v in prop::collection::vec(any::<i32>(), 0..100)) {
        assert_eq!(reverse(&v).len(), v.len());
    }

    #[test]
    fn test_sort_is_idempotent(mut v in prop::collection::vec(any::<i32>(), 0..100)) {
        v.sort();
        let sorted_once = v.clone();
        v.sort();
        assert_eq!(v, sorted_once); // Sorting twice = sorting once
    }

    #[test]
    fn test_parse_roundtrip(x in any::<f64>().prop_filter("finite", |x| x.is_finite())) {
        // Property: formatting then parsing gives back the same value
        let s = format!("{x}");
        let parsed: f64 = s.parse().unwrap();
        prop_assert!((x - parsed).abs() < f64::EPSILON);
    }
}
